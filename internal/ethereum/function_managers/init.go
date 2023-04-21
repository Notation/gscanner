package funcmanager

var Efm *ExponentFunctionManager
var Kfm *KeccakFunctionManager

func Init() {
	Efm = NewExponentFunctionManager()
	Kfm = NewKeccakFunctionManager()
}
