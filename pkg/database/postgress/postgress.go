package postgress

import (
	"errors"
	"fmt"
	"os"

	m "streaming/pkg/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var ErrConnectToPostgressql = errors.New("Ошибка при подключнии к базе данных")

type storage struct{
	db *gorm.DB
}

type testStorage struct{}
func (t testStorage) Save(conn m.Connection) error {
	fmt.Println(conn)
	return nil
}
func (t testStorage) GetConnsUseIdChannel(ich m.IdChannel) ([]m.Connection, error) {return nil, nil}
func (t testStorage) GetConnsUseName(name m.Name) ([]m.Connection, error) {return nil, nil}

var _ m.IPostgres = (*storage)(nil)

func New(cnf m.ConfPostgres) m.IPostgres {
	if cnf.TestMod {
		return testStorage{} 
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cnf.Host, cnf.User, cnf.Password, cnf.Database, cnf.Port, cnf.Sslmode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { 
		fmt.Println(ErrConnectToPostgressql)
		os.Exit(1)
	}
	return &storage{db}
}

func (s *storage) Save(conn m.Connection) error {return nil}

func (s *storage) GetConnsUseIdChannel(ich m.IdChannel) ([]m.Connection, error) {return nil, nil}

func (s *storage) GetConnsUseName(name m.Name) ([]m.Connection, error) {return nil, nil}
