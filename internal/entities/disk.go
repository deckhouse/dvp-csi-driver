package entities

import "k8s.io/apimachinery/pkg/api/resource"

type Disk struct {
	Name     string
	Capacity resource.Quantity
}
