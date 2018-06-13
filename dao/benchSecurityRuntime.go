package dao

import (
	"github.com/vmware/harbor/models"
)

func GetContainerIDCount(id string) (int, error) {
	sql := `select count(id) from bench_security_runTime where continer_id = ?`
	var num int
	err := GetOrmer().Raw(sql, id).QueryRow(&num)
	return num, err
}

func AddRuntime(id string, name string, runtime string, ip string) (int64, error) {
	o := GetOrmer()
	sql := `insert into bench_security_runTime(continer_id,continer_name,ip,container_runtime) values(?,?,?,?)`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(id, name, ip, runtime)
	if err != nil {
		return 0, err
	}
	pid, err := r.LastInsertId()
	return pid, err
}

func UpdateRuntime(id string, runtime string) error {
	o := GetOrmer()

	sql := `update bench_security_runTime set container_runtime = ? where continer_id = ?`

	_, err := o.Raw(sql, runtime, id).Exec()

	return err
}


func GetRunTimeById(id string) (string, error) {
	sql := `select container_runtime from bench_security_runTime where continer_id = ?`
	var runtime string
	err := GetOrmer().Raw(sql, id).QueryRow(&runtime)
	return runtime, err

}

func GetRunTimeByIp(ip string) ([]models.BenchSecurityRuntime, error) {
	sql := `select * from bench_security_runTime where ip = ?`
	var runtime []models.BenchSecurityRuntime
	_, err := GetOrmer().Raw(sql, ip).QueryRows(&runtime)
	return runtime, err

}

