package main
import (
  "fmt"
  "os"
  "os/exec"
  "runtime"
  "strings"
)
func main(){
  fmt.Println("GOOS:", runtime.GOOS)
  wd,_:=os.Getwd(); fmt.Println("PWD:", wd)
  if lp,err:=exec.LookPath("docker"); err!=nil { fmt.Println("LookPath ERR:", err) } else { fmt.Println("LookPath:", lp) }
  ps:=exec.Command("powershell","-NoProfile","-Command","(Get-Command docker -ErrorAction SilentlyContinue).Source")
  out,err:=ps.CombinedOutput(); fmt.Println("Get-Command err:", err); fmt.Println(strings.TrimSpace(string(out)))
  wh:=exec.Command("where.exe","docker")
  out2,err2:=wh.CombinedOutput(); fmt.Println("where err:", err2); fmt.Println(strings.TrimSpace(string(out2)))
}