package common

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"redis-plugs/models"
)

// GetKeyInfo 获取 Key 的详细信息
func GetKeyInfo(ctx context.Context, client *redis.Client, key string) (*models.KeyInfo, error) {
	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("获取 Key 类型失败: %w", err)
	}

	if keyType == "none" {
		return nil, fmt.Errorf("Key 不存在: %s", key)
	}

	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("获取 TTL 失败: %w", err)
	}

	info := &models.KeyInfo{
		Key:  key,
		Type: keyType,
		TTL:  int64(ttl.Seconds()),
	}

	switch keyType {
	case "string":
		err = getStringValue(ctx, client, key, info)
	case "list":
		err = getListValue(ctx, client, key, info)
	case "set":
		err = getSetValue(ctx, client, key, info)
	case "hash":
		err = getHashValue(ctx, client, key, info)
	case "zset":
		err = getZSetValue(ctx, client, key, info)
	default:
		info.Value = fmt.Sprintf("不支持的类型: %s", keyType)
	}

	return info, err
}

func getStringValue(ctx context.Context, client *redis.Client, key string, info *models.KeyInfo) error {
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	info.Value = val
	info.Size = int64(len(val))
	return nil
}

func getListValue(ctx context.Context, client *redis.Client, key string, info *models.KeyInfo) error {
	length, err := client.LLen(ctx, key).Result()
	if err != nil {
		return err
	}
	info.Size = length

	limit := int64(100)
	if length < limit {
		limit = length
	}

	val, err := client.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return err
	}
	info.Value = val
	return nil
}

func getSetValue(ctx context.Context, client *redis.Client, key string, info *models.KeyInfo) error {
	size, err := client.SCard(ctx, key).Result()
	if err != nil {
		return err
	}
	info.Size = size

	var members []string
	var cursor uint64 = 0

	for len(members) < 100 {
		vals, nextCursor, err := client.SScan(ctx, key, cursor, "*", 100).Result()
		if err != nil {
			return err
		}
		members = append(members, vals...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(members) > 100 {
		members = members[:100]
	}
	info.Value = members
	return nil
}

func getHashValue(ctx context.Context, client *redis.Client, key string, info *models.KeyInfo) error {
	size, err := client.HLen(ctx, key).Result()
	if err != nil {
		return err
	}
	info.Size = size

	result := make(map[string]string)
	var cursor uint64 = 0

	for len(result) < 100 {
		vals, nextCursor, err := client.HScan(ctx, key, cursor, "*", 200).Result()
		if err != nil {
			return err
		}

		for i := 0; i < len(vals)-1; i += 2 {
			if len(result) >= 100 {
				break
			}
			result[vals[i]] = vals[i+1]
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	info.Value = result
	return nil
}

func getZSetValue(ctx context.Context, client *redis.Client, key string, info *models.KeyInfo) error {
	size, err := client.ZCard(ctx, key).Result()
	if err != nil {
		return err
	}
	info.Size = size

	limit := int64(100)
	if size < limit {
		limit = size
	}

	vals, err := client.ZRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		return err
	}

	result := make([]map[string]interface{}, len(vals))
	for i, z := range vals {
		result[i] = map[string]interface{}{
			"member": z.Member,
			"score":  z.Score,
		}
	}
	info.Value = result
	return nil
}

// DeleteKey 删除 Key
func DeleteKey(ctx context.Context, client *redis.Client, key string) error {
	return client.Del(ctx, key).Err()
}
