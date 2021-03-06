package sql

import (
	"errors"
	"fmt"
	"log"
	"time"

	clouddriver "github.com/billiford/go-clouddriver/pkg"
	"github.com/billiford/go-clouddriver/pkg/kubernetes"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	// Needed for connection.
	_ "github.com/go-sql-driver/mysql"

	// Needed for connection.
	_ "github.com/mattn/go-sqlite3"
)

const (
	ClientInstanceKey = `SQLClient`
	maxOpenConns      = 5
	connMaxLifetime   = time.Second * 30
)

//go:generate counterfeiter . Client

type Client interface {
	GetKubernetesProvider(string) (kubernetes.Provider, error)
	CreateKubernetesProvider(kubernetes.Provider) error
	CreateKubernetesResource(kubernetes.Resource) error
	CreateReadPermission(clouddriver.ReadPermission) error
	CreateWritePermission(clouddriver.WritePermission) error
	ListKubernetesProviders() ([]kubernetes.Provider, error)
	ListKubernetesResourcesByFields(...string) ([]kubernetes.Resource, error)
	ListKubernetesResourcesByTaskID(string) ([]kubernetes.Resource, error)
	ListKubernetesAccountsBySpinnakerApp(string) ([]string, error)
	ListReadGroupsByAccountName(string) ([]string, error)
	ListWriteGroupsByAccountName(string) ([]string, error)
}

func NewClient(db *gorm.DB) Client {
	return &client{db: db}
}

type client struct {
	db *gorm.DB
}

// Connect sets up the database connection and creates tables.
//
// Connection is of type interface{} - this allows for tests to
// pass in a sqlmock connection and for main to connect given a
// connection string.
func Connect(driver string, connection interface{}) (*gorm.DB, error) {
	db, err := gorm.Open(driver, connection)
	if err != nil {
		return nil, err
	}

	db.LogMode(false)
	db.AutoMigrate(
		&kubernetes.Provider{},
		&kubernetes.Resource{},
		&clouddriver.ReadPermission{},
		&clouddriver.WritePermission{},
	)

	db.DB().SetMaxOpenConns(maxOpenConns)
	db.DB().SetMaxIdleConns(1)
	db.DB().SetConnMaxLifetime(connMaxLifetime)

	return db, nil
}

func Instance(c *gin.Context) Client {
	return c.MustGet(ClientInstanceKey).(Client)
}

type Config struct {
	User     string
	Password string
	Host     string
	Name     string
}

// Get driver and connection string to the DB.
func Connection(c Config) (string, string) {
	if c.User == "" || c.Password == "" || c.Host == "" || c.Name == "" {
		log.Println("SQL config missing field - defaulting to local sqlite DB.")
		return "sqlite3", "clouddriver.db"
	}

	return "mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=UTC",
		c.User, c.Password, c.Host, c.Name)
}

func (c *client) GetKubernetesProvider(name string) (kubernetes.Provider, error) {
	var p kubernetes.Provider
	db := c.db.Select("host, ca_data, bearer_token").Where("name = ?", name).First(&p)

	return p, db.Error
}

func (c *client) CreateKubernetesProvider(p kubernetes.Provider) error {
	db := c.db.Create(&p)
	return db.Error
}

func (c *client) CreateKubernetesResource(r kubernetes.Resource) error {
	db := c.db.Create(&r)
	return db.Error
}

func (c *client) CreateWritePermission(w clouddriver.WritePermission) error {
	db := c.db.Create(&w)
	return db.Error
}

func (c *client) CreateReadPermission(r clouddriver.ReadPermission) error {
	db := c.db.Create(&r)
	return db.Error
}

func (c *client) ListKubernetesProviders() ([]kubernetes.Provider, error) {
	var ps []kubernetes.Provider
	db := c.db.Select("name, host, ca_data").Find(&ps)

	return ps, db.Error
}

func (c *client) ListKubernetesResourcesByTaskID(taskID string) ([]kubernetes.Resource, error) {
	var rs []kubernetes.Resource
	db := c.db.Select("account_name, api_group, kind, name, namespace, resource, version").
		Where("task_id = ?", taskID).Find(&rs)

	return rs, db.Error
}

func (c *client) ListKubernetesResourcesByFields(fields ...string) ([]kubernetes.Resource, error) {
	if len(fields) == 0 {
		return nil, errors.New("no fields provided")
	}

	list := ""
	for i, field := range fields {
		list += field
		if i != len(fields)-1 {
			list += ", "
		}
	}

	var rs []kubernetes.Resource
	db := c.db.Select(list).Group(list).Find(&rs)

	return rs, db.Error
}

func (c *client) ListKubernetesAccountsBySpinnakerApp(spinnakerApp string) ([]string, error) {
	var rs []kubernetes.Resource
	db := c.db.Select("account_name").
		Where("spinnaker_app = ?", spinnakerApp).
		Group("account_name").
		Find(&rs)

	accounts := []string{}
	for _, r := range rs {
		accounts = append(accounts, r.AccountName)
	}

	return accounts, db.Error
}

func (c *client) ListReadGroupsByAccountName(accountName string) ([]string, error) {
	r := []clouddriver.ReadPermission{}
	db := c.db.Select("read_group").
		Where("account_name = ?", accountName).
		Group("read_group").
		Find(&r)

	groups := []string{}
	for _, v := range r {
		groups = append(groups, v.ReadGroup)
	}

	return groups, db.Error
}

func (c *client) ListWriteGroupsByAccountName(accountName string) ([]string, error) {
	w := []clouddriver.WritePermission{}
	db := c.db.Select("write_group").
		Where("account_name = ?", accountName).
		Group("write_group").
		Find(&w)

	groups := []string{}
	for _, v := range w {
		groups = append(groups, v.WriteGroup)
	}

	return groups, db.Error
}
