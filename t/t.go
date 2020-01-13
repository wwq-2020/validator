package t

type U struct {
	Name  string `validator:"required"`
	Value int    `validator:"required"`
	UU    *U
}
