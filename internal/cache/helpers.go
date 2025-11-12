package cache

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// addJitter 为 TTL 添加随机偏移，防止缓存雪崩
// 说明：在原始 TTL 基础上添加 ±5% 的随机偏移
// 参数：
//   - ttl: 原始 TTL（秒）
//
// 返回：
//   - 添加随机偏移后的 TTL（秒）
//
// 示例：
//   - ttl = 3600  → 返回 3420 ~ 3780（±180秒）
//   - ttl = 300   → 返回 285 ~ 315（±15秒）
func addJitter(ttl int) int {
	if ttl <= 0 {
		return ttl
	}

	// 计算 ±5% 的偏移量
	jitter := ttl / 20 // 5% = 1/20

	// 生成 [-jitter, +jitter] 范围的随机数
	offset := rand.Intn(2*jitter+1) - jitter

	return ttl + offset
}
