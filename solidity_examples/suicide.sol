pragma solidity 0.5.0;

contract Suicide {

  function kill(address payable addr) public {
    if (addr == address(0x0)) {
      selfdestruct(addr);
    }
  }

}
