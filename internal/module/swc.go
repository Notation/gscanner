package module

// 常见漏洞
// https://swcregistry.io/

type SWCData struct {
	ID          string
	Title       string
	Description string
}

var SWCDataMap = map[string]*SWCData{
	"104": {
		"104",
		"Unchecked Call Return Value",
		"The return value of a message call is not checked. Execution will resume even if the called contract throws an exception. If the call fails accidentally or an attacker forces the call to fail, this may cause unexpected behaviour in the subsequent program logic",
	},
	"106": {
		"106",
		"Unprotected SELFDESTRUCT Instruction",
		"Due to missing or insufficient access controls, malicious parties can self-destruct the contract.",
	},
	"107": {
		"107",
		"Reentrancy",
		"One of the major dangers of calling external contracts is that they can take over the control flow. In the reentrancy attack (a.k.a. recursive call attack), a malicious contract calls back into the calling contract before the first invocation of the function is finished. This may cause the different invocations of the function to interact in undesirable ways.",
	},
	"115": {
		"115",
		"Authorization through tx.origin",
		"tx.origin is a global variable in Solidity which returns the address of the account that sent the transaction. Using the variable for authorization could make a contract vulnerable if an authorized account calls into a malicious contract. A call could be made to the vulnerable contract that passes the authorization check since tx.origin returns the original sender of the transaction which in this case is the authorized account.",
	},
	"127": {
		"127",
		"Arbitrary Jump with Function Type Variable",
		"Solidity supports function types. That is, a variable of function type can be assigned with a reference to a function with a matching signature. The function saved to such variable can be called just like a regular function. The problem arises when a user has the ability to arbitrarily change the function type variable and thus execute random code instructions. As Solidity doesn't support pointer arithmetics, it's impossible to change such variable to an arbitrary value. However, if the developer uses assembly instructions, such as mstore or assign operator, in the worst case scenario an attacker is able to point a function type variable to any code instruction, violating required validations and required state changes.",
	},
}
