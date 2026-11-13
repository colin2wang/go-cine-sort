package netclient

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCalculateNonce(t *testing.T) {
	for _, challenge := range []string{"abc123", "douban-challenge-2024", ""} {
		sol, err := CalculateNonce(context.Background(), challenge, 4)
		if err != nil {
			t.Fatalf("challenge %q: %v", challenge, err)
		}
		sum := sha512.Sum512([]byte(challenge + sol))
		digest := hex.EncodeToString(sum[:])
		if !strings.HasPrefix(digest, "0000") {
			t.Fatalf("challenge %q: nonce %s 不满足前导 4 个 0: %s", challenge, sol, digest[:16])
		}
	}
}

func TestCalculateNonceCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CalculateNonce(ctx, "abc", 4); err == nil {
		t.Fatal("已取消的 context 应返回错误")
	}
}

func TestPowFormParsing(t *testing.T) {
	page := `<html><form name="sec" id="sec" action="/j/verify">
<input type="hidden" name="tok" value="TOKEN-1" />
<input type="hidden" name="cha" value="CHALLENGE-1" />
<input type="hidden" name="red" value="https://movie.douban.com/subject/1292052/" />
</form></html>`
	cases := []struct {
		re   func(string) (string, bool)
		want string
	}{
		{func(s string) (string, bool) { return firstMatch(rePowTok, s) }, "TOKEN-1"},
		{func(s string) (string, bool) { return firstMatch(rePowCha, s) }, "CHALLENGE-1"},
		{func(s string) (string, bool) { return firstMatch(rePowRed, s) }, "https://movie.douban.com/subject/1292052/"},
		{func(s string) (string, bool) { return firstMatch(rePowAction, s) }, "/j/verify"},
	}
	for i, c := range cases {
		got, ok := c.re(page)
		if !ok || got != c.want {
			t.Errorf("case %d: got %q (ok=%v), want %q", i, got, ok, c.want)
		}
	}
}
