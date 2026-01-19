package common

import (
	"context"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
	"redis-plugs/config"
	"redis-plugs/models"
)

// ScanAllKeys 使用 SCAN 命令扫描所有 Key（避免使用 KEYS *）
func ScanAllKeys(ctx context.Context, client *redis.Client, pattern string, maxKeys int64) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	if maxKeys <= 0 {
		maxKeys = config.DefaultScanConfig.MaxCount
	}

	var allKeys []string
	var cursor uint64 = 0

	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, config.DefaultScanConfig.DefaultCount).Result()
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)
		cursor = nextCursor

		if cursor == 0 || int64(len(allKeys)) >= maxKeys {
			break
		}
	}

	if int64(len(allKeys)) > maxKeys {
		allKeys = allKeys[:maxKeys]
	}

	return allKeys, nil
}

// GetKeyTree 获取 Key 树（懒加载模式，只获取一层，每层最多10个）
// prefix: 当前路径前缀，用于解析下一层子节点
func GetKeyTree(ctx context.Context, client *redis.Client, pattern, separator, prefix string, maxKeys int64) (*models.KeyNode, error) {
	keys, err := ScanAllKeys(ctx, client, pattern, maxKeys)
	if err != nil {
		return nil, err
	}

	// 如果有前缀，还需要检查前缀本身是否是一个有效的 key
	if prefix != "" {
		exists, err := client.Exists(ctx, prefix).Result()
		if err == nil && exists > 0 {
			keys = append(keys, prefix)
		}
	}

	return buildKeyTreeOneLevel(keys, separator, prefix, 10), nil
}

// SearchKeyTree 模糊搜索模式，返回匹配的 key 树（完整构建）
func SearchKeyTree(ctx context.Context, client *redis.Client, pattern, separator string, maxKeys int64) (*models.KeyNode, error) {
	keys, err := ScanAllKeys(ctx, client, pattern, maxKeys)
	if err != nil {
		return nil, err
	}
	return buildSearchTree(keys, separator, 10), nil
}

// buildSearchTree 构建搜索结果树（从头构建完整路径，每层限制数量）
func buildSearchTree(keys []string, separator string, maxChildren int) *models.KeyNode {
	if separator == "" {
		separator = ":"
	}

	root := &models.KeyNode{
		Name:     "root",
		Children: make([]*models.KeyNode, 0),
	}

	for _, key := range keys {
		insertKeyToTree(root, key, separator)
	}

	// 限制每层子节点数量并排序
	limitAndSortTree(root, maxChildren)

	return root
}

// insertKeyToTree 将 key 插入树中
func insertKeyToTree(root *models.KeyNode, key, separator string) {
	parts := strings.Split(key, separator)
	current := root

	for i, part := range parts {
		isLast := i == len(parts)-1
		child := findChild(current, part)

		if child == nil {
			child = &models.KeyNode{
				Name:     part,
				Children: make([]*models.KeyNode, 0),
				Count:    0,
			}
			current.Children = append(current.Children, child)
		}

		if isLast {
			child.IsLeaf = true
			child.FullKey = key
		}
		child.Count++
		current = child
	}
}

// findChild 查找子节点
func findChild(node *models.KeyNode, name string) *models.KeyNode {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// limitAndSortTree 递归限制并排序树
func limitAndSortTree(node *models.KeyNode, maxChildren int) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	sortChildren(node)

	if len(node.Children) > maxChildren {
		node.Children = node.Children[:maxChildren]
	}

	for _, child := range node.Children {
		limitAndSortTree(child, maxChildren)
	}

	node.Count = len(node.Children)
}

// buildKeyTreeOneLevel 构建单层树结构（懒加载模式）
func buildKeyTreeOneLevel(keys []string, separator, prefix string, maxChildren int) *models.KeyNode {
	if separator == "" {
		separator = config.DefaultScanConfig.DefaultSeparator
	}

	root := &models.KeyNode{
		Name:     "root",
		Children: make([]*models.KeyNode, 0),
	}

	// 用于记录每个子节点的信息
	type childInfo struct {
		node     *models.KeyNode
		hasChild bool // 是否有更深层的子节点
	}
	childMap := make(map[string]*childInfo)

	for _, key := range keys {
		// 处理 prefix 本身就是一个 key 的情况
		if key == prefix {
			root.FullKey = prefix
			continue
		}

		// 获取相对于前缀的剩余部分
		var remaining string
		if prefix == "" {
			remaining = key
		} else if strings.HasPrefix(key, prefix+separator) {
			remaining = strings.TrimPrefix(key, prefix+separator)
		} else {
			continue
		}

		// 获取第一层的名称
		parts := strings.SplitN(remaining, separator, 2)
		firstName := parts[0]
		if firstName == "" {
			continue
		}

		hasMoreLevels := len(parts) > 1

		// 检查是否已存在该子节点
		if info, exists := childMap[firstName]; exists {
			info.node.Count++
			if hasMoreLevels {
				info.hasChild = true
			}
		} else {
			fullKey := ""
			if prefix == "" {
				fullKey = firstName
			} else {
				fullKey = prefix + separator + firstName
			}
			child := &models.KeyNode{
				Name:     firstName,
				FullKey:  fullKey,
				IsLeaf:   true,
				Children: make([]*models.KeyNode, 0),
				Count:    1,
			}
			childMap[firstName] = &childInfo{
				node:     child,
				hasChild: hasMoreLevels,
			}
		}
	}

	// 转换为切片，并根据 hasChild 修正 IsLeaf
	for _, info := range childMap {
		if info.hasChild {
			info.node.IsLeaf = false
			info.node.FullKey = ""
		}
		root.Children = append(root.Children, info.node)
	}

	sortChildren(root)

	// 限制子节点数量
	if len(root.Children) > maxChildren {
		root.Children = root.Children[:maxChildren]
	}

	root.Count = len(root.Children)

	// 如果 root 有 fullKey 且没有子节点，说明它本身就是叶子节点
	if root.FullKey != "" && len(root.Children) == 0 {
		root.IsLeaf = true
	}

	return root
}

func sortChildren(node *models.KeyNode) {
	if len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		if !node.Children[i].IsLeaf && node.Children[j].IsLeaf {
			return true
		}
		if node.Children[i].IsLeaf && !node.Children[j].IsLeaf {
			return false
		}
		return node.Children[i].Name < node.Children[j].Name
	})
}
