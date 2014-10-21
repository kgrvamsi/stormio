package persistence

import (
	log "github.com/cihub/seelog"
	"github.com/coocood/qbs"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type AssetDS struct {
	conn *qbs.Qbs
}

func (as *AssetDS) GetConnection() (conn *qbs.Qbs, err error) {
	for i := 0; i < 5; i++ {
		conn, err = qbs.GetQbs()
		if err != nil {
			log.Debugf("Error in getting db connection %v... retrying..", err)
			time.Sleep(time.Duration(10) * time.Second)
			continue
		} else {
			break
		}
	}
	return
}

func (as *AssetDS) Close() {
	as.conn.Close()
}

func (as *AssetDS) InitDatabase(provider, dbname, username, password, host, port string) error {
	// ("mysql", "cloudio:password@tcp(localhost:3306)/cloudio")
	dsn := &qbs.DataSourceName{Dialect: qbs.NewMysql(), DbName: dbname, Username: username, Password: password, Host: host, Port: port}
	qbs.RegisterWithDataSourceName(dsn)
	q, err := qbs.GetQbs()
	defer q.Close()
	return err
}

func (as *AssetDS) Create(conn *qbs.Qbs, ar interface{}) error {
	_, err := conn.Save(ar)
	return err
}

func (as *AssetDS) Update(conn *qbs.Qbs, ar interface{}) error {
	_, err := conn.Update(ar)
	return err
}

func (as *AssetDS) Remove(conn *qbs.Qbs, id string) error {
	ar, err := as.Find(conn, id)
	if err != nil {
		log.Debug("Asset request not found :%s", id)
		return err
	}
	ms := &ModuleStatus{AssetRequestId: id}
	conn.WhereEqual("asset_request_id", id).Delete(ms)
	_, err = conn.Delete(ar)
	return err
}

func (as *AssetDS) FindModuleStatus(conn *qbs.Qbs, resourceId, name string) (*ModuleStatus, error) {
	ar := new(AssetRequest)
	err := conn.WhereEqual("resource_id", resourceId).Find(ar)
	ms := new(ModuleStatus)
	if err == nil {
		condition := qbs.NewCondition("asset_request_id = ?", ar.Id).And("name =? ", name)
		err = conn.Condition(condition).Find(ms)
	}

	return ms, err
}

func (as *AssetDS) FindBy(fieldName string, conn *qbs.Qbs, resourceId string) (*AssetRequest, error) {
	ar := new(AssetRequest)
	err := conn.WhereEqual(fieldName, resourceId).Find(ar)
	if mss, err := loadModules(conn, ar.Id); err == nil {
		ar.Modules = mss
	}
	return ar, err
}

func (as *AssetDS) Find(conn *qbs.Qbs, id string) (*AssetRequest, error) {
	ar := new(AssetRequest)
	ar.Id = id
	err := conn.Find(ar)
	if mss, err := loadModules(conn, id); err == nil {
		ar.Modules = mss
	}
	return ar, err
}

func loadModules(conn *qbs.Qbs, arId string) ([]ModuleStatus, error) {
	var mss []ModuleStatus
	err := conn.WhereEqual("asset_request_id", arId).FindAll(&mss)
	return mss, err
}

func (as *AssetDS) FindAllBy(field string, conn *qbs.Qbs, status ...interface{}) ([]*AssetRequest, error) {
	var ars []*AssetRequest
	err := conn.WhereIn(field, status).FindAll(&ars)
	return ars, err
}
