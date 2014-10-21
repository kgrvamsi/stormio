package persistence

import (
	"stormstack.org/cloudio/util"
	"labix.org/v2/mgo"
)

type Connection struct {
	session    *mgo.Session
	collection *mgo.Collection
}

const (
	Database         = "CloudIO"
	Collection       = "Assets"
	CFCollectionName = "ConfigPassThru"
)

func DefaultSession() (conn *Connection, err error) {
	return NewConnection(Database, Collection)
}

func NewConnection(dbName, collName string) (conn *Connection, err error) {
	// log.Debugf("Opening mongo session")
	conn = new(Connection)
	session, err := mgo.Dial(util.GetString("database", "host") + ":" + util.GetString("database", "port"))
	if err != nil {
		return
	}
	conn.session = session
	conn.collection = conn.session.DB(dbName).C(collName)
	return
}

func (conn *Connection) GetCollection() (collection *mgo.Collection) {
	collection = conn.collection
	return
}

func (conn *Connection) Close() {
	// log.Debugf("Closed the session")
	conn.session.Close()
}

func (conn *Connection) Create(assetReq *AssetRequest) (err error) {
	return conn.collection.Insert(assetReq)
}

func (conn *Connection) Update(assetReq *AssetRequest) (err error) {
	conn.collection.UpsertId(assetReq.Id, assetReq)
	return err
}

func (conn *Connection) Remove(id string) error {
	err := conn.collection.RemoveId(id)
	return err
}

func (conn *Connection) GenericFind(obj, criteria interface{}) (err error) {
	err = conn.collection.Find(criteria).One(obj)
	return
}

func (conn *Connection) Find(criteria interface{}) (assetReq *AssetRequest, err error) {
	_assetReq := new(AssetRequest)
	err = conn.collection.Find(criteria).One(_assetReq)
	if err == nil {
		assetReq = _assetReq
	}
	return
}

func (conn *Connection) FindAll(criteria interface{}) (assets []*AssetRequest, err error) {
	var _assets []*AssetRequest
	err = conn.collection.Find(criteria).All(&_assets)

	if err == nil {
		assets = _assets
	}
	return
}
