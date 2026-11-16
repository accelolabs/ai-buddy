package users

const (
	RoleIntern = "intern"
	RoleMentor = "mentor"
)

type User struct {
	ID   string
	Name string
	Role string
	Team string
}
