package mask

import "testing"

var sinkErr error

type benchUser struct {
	Name     string `mask:"name"`
	Phone    string `mask:"phone"`
	IDCard   string `mask:"idcard"`
	Email    string `mask:"email"`
	Password string `mask:"password"`
	Address  string `mask:"address"`
	Note     string
}

func BenchmarkMaskSimple(b *testing.B) {
	u := benchUser{Name: "张三", Phone: "13800138000", IDCard: "110101199003071234",
		Email: "zhangsan@example.com", Password: "secret", Address: "北京市朝阳区", Note: "n"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkErr = Mask(&u)
	}
}

type benchCard struct {
	No string `mask:"bankcard"`
}
type benchHolder struct {
	User benchUser
	Card benchCard
}

func BenchmarkMaskNested(b *testing.B) {
	h := benchHolder{
		User: benchUser{Name: "李四", Phone: "13900139000"},
		Card: benchCard{No: "6222020200112233"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkErr = Mask(&h)
	}
}

func BenchmarkMaskSlice(b *testing.B) {
	users := make([]benchUser, 100)
	for i := range users {
		users[i] = benchUser{Name: "张三", Phone: "13800138000"}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkErr = Mask(&users)
	}
}
