package main

const (
	TOKEN_TEST1 = 256 + iota
	TOKEN_TEST2
)

const (
	STATE_INIT = iota
)

var (
	rules = map[string][]*Rule{
		{
			"_INIT": {
				"test1",
				action_input,
			}
		}
	}
)

func action_test1(value string) error {
	fm.Println("TEST1")
	return nil
}

func action_test2(value string) error {
	fm.Println("TEST2")
	return nil
}
