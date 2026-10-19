package xray
import "testing"
func TestParseTrafficStat(t *testing.T){
 tests:=[]struct{name string; inbound bool; tag string; down bool; ok bool}{
 {"inbound>>>inbound-443>>>traffic>>>uplink",true,"inbound-443",false,true},
 {"inbound>>>inbound-443>>>traffic>>>downlink",true,"inbound-443",true,true},
 {"outbound>>>api>>>traffic>>>uplink",false,"api",false,true},
 {"not-a-traffic-stat",false,"",false,false},
 {"inbound>>>broken>>>traffic",false,"",false,false},
 }
 for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){i,tag,d,ok:=parseTrafficStat(tt.name);if i!=tt.inbound||tag!=tt.tag||d!=tt.down||ok!=tt.ok{t.Fatalf("got (%v,%q,%v,%v), want (%v,%q,%v,%v)",i,tag,d,ok,tt.inbound,tt.tag,tt.down,tt.ok)}})}
}
