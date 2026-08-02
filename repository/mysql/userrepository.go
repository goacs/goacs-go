package mysql

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"goacs/models/user"
	"goacs/repository"
	"golang.org/x/crypto/bcrypt"
	"log"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(connection *sqlx.DB) *UserRepository {
	return &UserRepository{
		db: connection,
	}
}

func (r *UserRepository) Find(uuid string) (user.User, error) {
	var userModel user.User
	err := r.db.Get(&userModel, "SELECT * FROM users WHERE uuid=?", &uuid)
	return userModel, err
}

func (r *UserRepository) GetUserByAuthData(username string, password string) (user.User, error) {
	var userModel user.User
	err := r.db.Get(&userModel, "SELECT * FROM users WHERE username=?", &username)

	if err != nil {
		return user.User{}, err
	}

	log.Println(userModel.Password, password, user.EncryptPassword(password))

	err = bcrypt.CompareHashAndPassword([]byte(userModel.Password), []byte(password))

	if err != nil {
		return user.User{}, err
	}

	return userModel, nil

}

func (r *UserRepository) CreateUser(userModel *user.User) (user.User, error) {
	uuidInstance, _ := uuid.NewRandom()
	userModel.Uuid = uuidInstance.String()
	userModel.Status = 1
	_, err := r.db.Exec("INSERT INTO users VALUES (?,?,?,?,?)",
		userModel.Uuid, userModel.Username, userModel.Password, userModel.Email, userModel.Status)

	return *userModel, err
}

func (r *UserRepository) List(request repository.PaginatorRequest) ([]user.User, int) {
	var total int
	var users = make([]user.User, 0)

	_ = r.db.Get(&total, "SELECT count(*) FROM users")
	_ = r.db.Select(&users, "SELECT * FROM users LIMIT ?,?", request.CalcOffset(), request.PerPage)

	return users, total
}

// Update sets username/email unconditionally, and password only when a
// non-empty (already-hashed) value is supplied, mirroring goacs-php's
// UserController::update, which never forces an admin to re-enter a password.
func (r *UserRepository) Update(userModel *user.User, hashedPassword string) error {
	dialect := goqu.Dialect("mysql")

	record := goqu.Record{
		"username": userModel.Username,
		"email":    userModel.Email,
	}

	if hashedPassword != "" {
		record["password"] = hashedPassword
	}

	query, args, _ := dialect.Update("users").Prepared(true).
		Set(record).
		Where(goqu.C("uuid").Eq(userModel.Uuid)).
		ToSQL()

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *UserRepository) Delete(uuid string) error {
	dialect := goqu.Dialect("mysql")

	query, args, _ := dialect.Delete("users").Prepared(true).
		Where(goqu.C("uuid").Eq(uuid)).
		ToSQL()

	_, err := r.db.Exec(query, args...)
	return err
}
