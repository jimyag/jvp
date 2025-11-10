// Package idgen 提供递增 ID 生成器
//
// 使用 Sonyflake 算法生成全局唯一且递增的 ID。
// Sonyflake 是 Snowflake 算法的改进版本，生成的 ID 具有以下特性：
//   - 全局唯一
//   - 时间有序（递增）
//   - 64 位整数
//   - 分布式友好
//
// 生成的 ID 格式：
//   - 镜像 ID: ami-{递增数字}
//   - Volume ID: vol-{递增数字}
//   - Instance ID: i-{递增数字}
//
// 使用方式：
//
// 方式一：使用包级别的便捷函数（推荐，使用默认生成器）
//
//	// 生成镜像 ID
//	imageID, err := idgen.GenerateImageID()
//	// imageID: "ami-1234567890"
//
//	// 生成 Volume ID
//	volumeID, err := idgen.GenerateVolumeID()
//	// volumeID: "vol-1234567891"
//
//	// 生成 Instance ID
//	instanceID, err := idgen.GenerateInstanceID()
//	// instanceID: "i-1234567892"
//
// 方式二：使用默认生成器
//
//	gen := idgen.DefaultGenerator()
//	imageID, err := gen.GenerateImageID()
//
// 方式三：创建自定义生成器
//
//	gen := idgen.New()
//	imageID, err := gen.GenerateImageID()
package idgen
