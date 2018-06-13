package dao

func UpdateAdminTime(admintype string, t int) error {
	o := GetOrmer()
	sql := "update properties set v = ? where k = ? ;"

	_, err := o.Raw(sql, t, admintype).Exec()
	return err
}

func GetAdminTime(admintype string) (int, error) {
	sql := `select v from properties where k = ? ;`
	var num int
	err := GetOrmer().Raw(sql, admintype).QueryRow(&num)
	return num, err
}
