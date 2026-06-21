package surveyerr

import "fmt"

var ErrCode = -1
var ErrNotFound = fmt.Errorf("entry not found")
var ErrNodeCountCannotLessThenOne = fmt.Errorf("Must be at least 1 node")
