package videosource

import (
	"fmt"
	"image"
	"sort"
	"testing"
	"time"
)

func TestProcessedImageContentSort(t *testing.T) {
	obj1 := NewObjectInfo(image.Rectangle{}, *NewColorThickness("blue", 1))
	obj1.Percentage = 80
	obj2 := NewObjectInfo(image.Rectangle{}, *NewColorThickness("blue", 1))
	obj2.Percentage = 90

	face1 := NewFaceInfo(image.Rectangle{}, *NewColorThickness("green", 1))
	face1.Percentage = 80
	face2 := NewFaceInfo(image.Rectangle{}, *NewColorThickness("green", 1))
	face2.Percentage = 90

	now := time.Now()
	first := NewProcessedImage(Image{CreatedTime: now.Add(time.Hour)})
	first.Objects = append(first.Objects, *obj1)
	first.Objects = append(first.Objects, *obj1)
	first.Faces = append(first.Faces, *face1)
	first.Faces = append(first.Faces, *face1)

	second := NewProcessedImage(Image{CreatedTime: now})
	second.Objects = append(second.Objects, *obj2)
	second.Faces = append(second.Faces, *face2)

	list := make([]ProcessedImage, 0)
	list = append(list, *first)
	list = append(list, *second)

	for i, cur := range list {
		fmt.Printf("%d - %v\n", i, cur.Original.CreatedTime)
		fmt.Printf("\tObjects: %d\n", len(cur.Objects))
		for j, obj := range cur.Objects {
			fmt.Printf("\t\t%d - Percent %d\n", j, obj.Percentage)
		}
		fmt.Printf("\tFaces: %d\n", len(cur.Faces))
		for j, face := range cur.Faces {
			fmt.Printf("\t\t%d - Percent %d\n", j, face.Percentage)
		}
	}
	sort.Sort(ProcessedImageByObjLen(list))
	sort.Sort(ProcessedImageByObjPercent(list))
	sort.Sort(ProcessedImageByFaceLen(list))
	sort.Sort(ProcessedImageByFacePercent(list))
	fmt.Println("Sorted")
	for i, cur := range list {
		fmt.Printf("%d - %v\n", i, cur.Original.CreatedTime)
		fmt.Printf("\tObjects: %d\n", len(cur.Objects))
		for j, obj := range cur.Objects {
			fmt.Printf("\t\t%d - Percent %d\n", j, obj.Percentage)
		}
		fmt.Printf("\tFaces: %d\n", len(cur.Faces))
		for j, face := range cur.Faces {
			fmt.Printf("\t\t%d - Percent %d\n", j, face.Percentage)
		}
	}
}
