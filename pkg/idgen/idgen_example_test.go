package idgen_test

import (
	"fmt"

	"github.com/jimyag/jvp/pkg/idgen"
)

func ExampleGenerator_GenerateImageID() {
	gen := idgen.New()

	// 生成镜像 ID
	imageID, err := gen.GenerateImageID()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 验证格式
	if len(imageID) > 4 && imageID[:4] == "ami-" {
		fmt.Println("Image ID format is correct")
	}
	// Output: Image ID format is correct
}

func ExampleGenerator_GenerateVolumeID() {
	gen := idgen.New()

	// 生成 Volume ID
	volumeID, err := gen.GenerateVolumeID()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 验证格式
	if len(volumeID) > 4 && volumeID[:4] == "vol-" {
		fmt.Println("Volume ID format is correct")
	}
	// Output: Volume ID format is correct
}

func ExampleGenerator_GenerateInstanceID() {
	gen := idgen.New()

	// 生成 Instance ID
	instanceID, err := gen.GenerateInstanceID()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 验证格式
	if len(instanceID) > 2 && instanceID[:2] == "i-" {
		fmt.Println("Instance ID format is correct")
	}
	// Output: Instance ID format is correct
}

func ExampleGenerator_GenerateID() {
	gen := idgen.New()

	// 生成多个 ID，验证它们是递增的
	var prevID uint64
	for i := 0; i < 5; i++ {
		id, err := gen.GenerateID()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if i > 0 && id > prevID {
			fmt.Printf("ID %d is greater than previous ID\n", i+1)
		}
		prevID = id
	}
	// Output:
	// ID 2 is greater than previous ID
	// ID 3 is greater than previous ID
	// ID 4 is greater than previous ID
	// ID 5 is greater than previous ID
}

func ExampleDefaultGenerator() {
	// 使用默认生成器
	gen := idgen.DefaultGenerator()

	imageID, err := gen.GenerateImageID()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(imageID) > 4 && imageID[:4] == "ami-" {
		fmt.Println("Using default generator")
	}
	// Output: Using default generator
}

func ExampleGenerateImageID() {
	// 使用包级别的便捷函数，直接使用默认生成器
	imageID, err := idgen.GenerateImageID()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(imageID) > 4 && imageID[:4] == "ami-" {
		fmt.Println("Using package-level function")
	}
	// Output: Using package-level function
}
