package models

type BenchSecurityRuntime struct {
	Id	          int    `orm:"pk;column(id)" json:"id"`
	Continer_id       string `orm:"column(continer_id)" json:"continer_id"`
	Continer_name     string `orm:"column(continer_name)" json:"continer_name"`
	Ip                string `orm:"column(ip)" json:"ip"`
	Container_runtime string `orm:"column(container_runtime)" json:"container_runtime"`
}

