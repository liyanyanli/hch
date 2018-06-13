package dao

import (
	"github.com/vmware/harbor/models"
)

func GetIpCount(ip string) (int, error) {
	sql := `select count(ip) from bench_security where ip = ?`
	var num int
	err := GetOrmer().Raw(sql, ip).QueryRow(&num)
	return num, err
}

func AddBenchResult(ip string, host string, daemo string, daemoFile string, images string, operations string) (int64, error) {
	o := GetOrmer()
	sql := `insert into bench_security(ip,host_configuration,docker_daemon_configuration,docker_daemon_configuration_files,container_images_and_build_files,docker_security_operations) values(?,?,?,?,?,?);`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(ip, host, daemo, daemoFile, images, operations)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	return id, err
}

func UpdateBenchResult(ip string, host string, daemo string, daemoFile string, images string, operations string) error {
	o := GetOrmer()

	sql := "update bench_security set host_configuration = ?, docker_daemon_configuration = ?, docker_daemon_configuration_files = ?, container_images_and_build_files = ?, docker_security_operations = ? where ip = ?"

	_, err := o.Raw(sql, host, daemo, daemoFile, images, operations, ip).Exec()

	return err
}

func GetAllBench () ([]models.BenchSecurity, error) {
	o := GetOrmer()
	sql := `select * from bench_security`

	var result []models.BenchSecurity

	if _, err := o.Raw(sql).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

