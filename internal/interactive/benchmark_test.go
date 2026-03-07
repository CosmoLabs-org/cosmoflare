package interactive

import (
	"testing"
	"time"
)

func BenchmarkEncryptBackupData(b *testing.B) {
	bm := &BackupManager{}
	data := &BackupData{
		Version:   "1.0",
		CreatedAt: time.Now(),
		Profiles: map[string]BackupProfile{
			"prod": {Name: "prod", AccountID: "abc123", Region: "us-east-1"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm.encryptBackupData(data, "benchmark-password")
	}
}

func BenchmarkDecryptBackupData(b *testing.B) {
	bm := &BackupManager{}
	data := &BackupData{
		Version:   "1.0",
		CreatedAt: time.Now(),
		Profiles: map[string]BackupProfile{
			"prod": {Name: "prod", AccountID: "abc123"},
		},
	}

	encrypted, _ := bm.encryptBackupData(data, "benchmark-password")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bm.decryptBackupData(encrypted, "benchmark-password")
	}
}

func BenchmarkValidateAccountID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateAccountID("abcdef0123456789abcdef0123456789")
	}
}
