package models

type BenchSecurity struct {
	Id	   int    `orm:"pk;column(id)" json:"id"`
	//UserName   string `orm:"column(user_name)" json:"userName"`
	Ip         string `orm:"column(ip)" json:"ip"`
	Host       string `orm:"column(host_configuration)" json:"host"`
	Daemon     string `orm:"column(docker_daemon_configuration)" json:"daemon"`
	DaemonFile string `orm:"column(docker_daemon_configuration_files)" json:"daemonFile"`
	Images     string `orm:"column(container_images_and_build_files)" json:"images"`
	//Runtime    string `orm:"column(container_runtime)" json:"runtime"`
	Operation  string `orm:"column(docker_security_operations)" json:"operation"`
}
