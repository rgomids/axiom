package main
import("bytes";"fmt";"os";"path/filepath";"github.com/rgomids/axiom/internal/workflow")
func main(){
 for _,path:=range os.Args[1:] {
 wire,e:=os.ReadFile(path);if e!=nil{panic(e)}
 decoded,e:=workflow.DecodeExecution(wire);if e!=nil{panic(e)}
 back,e:=workflow.EncodeExecution(decoded);if e!=nil||!bytes.Equal(back,wire){panic("old bytes changed")}
 for _,field:=range []string{`"attemptPolicy":"spawn-boundary-v1",`,`"launchFailure":{"reason":"builder_failed","phase":"invocation_build"},`} {
 candidate:=bytes.Replace(wire,[]byte(`"attemptId":`),[]byte(field+`"attemptId":`),1)
 if field[1]=='a'{candidate=append([]byte("{"+field),wire[1:]...)}
 effects:=0
 if _,e=workflow.DecodeExecution(candidate);e==nil {effects++;panic("old reader accepted new field")}
 if effects!=0{panic("effects before admission")}
 }
 fmt.Println(filepath.Base(path),"old reader accepts original; rejects candidate policy/failure before admission")
 }
}
