package helper

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrTemporaryDelivery - kegagalan kirim yang layak dicoba ulang (retry).
var ErrTemporaryDelivery = errors.New("temporary delivery failure")

// ErrPermanentDelivery - kegagalan permanen; retry tidak akan membantu.
var ErrPermanentDelivery = errors.New("permanent delivery failure")

// Mailer - kontrak pengiriman pesan. Diganti oleh fake mailer di test.
type Mailer interface {
	Send(to, subject, body string) error
}

// LogMailer meniru pengiriman email dengan menulis ke logs/mail.log.
// Delay mensimulasikan latensi jaringan; failEvery memicu kegagalan
// sementara agar perilaku retry bisa diamati.
type LogMailer struct {
	delay     time.Duration
	failEvery int
	logPath   string

	mu    sync.Mutex
	count int
}

// NewLogMailer membuat mailer berbasis file log.
// delay: latensi per kirim; failEvery: setiap N kirim gagal sementara (0 = tidak pernah).
func NewLogMailer(delay time.Duration, failEvery int, logPath string) *LogMailer {
	return &LogMailer{
		delay:     delay,
		failEvery: failEvery,
		logPath:   logPath,
	}
}

// Send menulis satu baris log. Gagal sementara bila hitungan kelipatan failEvery.
func (m *LogMailer) Send(to, subject, body string) error {
	m.mu.Lock()
	m.count++
	n := m.count
	m.mu.Unlock()

	// Simulasi latensi jaringan.
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Simulasi kegagalan sementara tiap failEvery-kirim.
	if m.failEvery > 0 && n%m.failEvery == 0 {
		err := fmt.Errorf("%w: smtp unavailable on attempt %d", ErrTemporaryDelivery, n)
		m.writeLog("FAIL", to, subject, err.Error())
		return err
	}

	m.writeLog("OK", to, subject, "")
	return nil
}

func (m *LogMailer) writeLog(status, to, subject, detail string) {
	if dir := filepath.Dir(m.logPath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}

	f, err := os.OpenFile(m.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("mailer: gagal membuka log", slog.String("error", err.Error()))
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s | %-4s | to=%s | subject=%q", time.Now().Format(time.RFC3339), status, to, subject)
	if detail != "" {
		line += " | " + detail
	}
	line += "\n"

	if _, err := f.WriteString(line); err != nil {
		slog.Error("mailer: gagal menulis log", slog.String("error", err.Error()))
	}
}
