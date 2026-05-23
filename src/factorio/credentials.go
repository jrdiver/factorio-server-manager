package factorio

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Credentials struct {
	Username string `json:"username"`
	Userkey  string `json:"userkey"`
}

type credentialRecord struct {
	gorm.Model
	Username         string
	EncryptedUserkey string
}

func getCredentialDB() (*gorm.DB, error) {
	config := bootstrap.GetConfig()
	db, err := gorm.Open(sqlite.Open(config.SQLiteDatabaseFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&credentialRecord{}); err != nil {
		return nil, err
	}
	return db, nil
}

func getEncryptionKey() ([]byte, error) {
	config := bootstrap.GetConfig()
	key, err := base64.StdEncoding.DecodeString(config.CookieEncryptionKey)
	if err != nil {
		return nil, err
	}
	if len(key) < 16 {
		return nil, errors.New("encryption key too short for AES")
	}
	if len(key) > 32 {
		key = key[:32]
	}
	return key, nil
}

func encryptValue(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptValue(key []byte, encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (credentials *Credentials) Save() error {
	db, err := getCredentialDB()
	if err != nil {
		log.Printf("error opening credential db: %s", err)
		return err
	}

	key, err := getEncryptionKey()
	if err != nil {
		log.Printf("error getting encryption key: %s", err)
		return err
	}

	encryptedKey, err := encryptValue(key, credentials.Userkey)
	if err != nil {
		log.Printf("error encrypting userkey: %s", err)
		return err
	}

	var record credentialRecord
	result := db.First(&record)
	if result.Error != nil {
		record = credentialRecord{
			Username:         credentials.Username,
			EncryptedUserkey: encryptedKey,
		}
		return db.Create(&record).Error
	}
	return db.Model(&record).Updates(map[string]interface{}{
		"username":          credentials.Username,
		"encrypted_userkey": encryptedKey,
	}).Error
}

func (credentials *Credentials) Load() (bool, error) {
	db, err := getCredentialDB()
	if err != nil {
		log.Printf("error opening credential db: %s", err)
		return false, err
	}

	var record credentialRecord
	if err := db.First(&record).Error; err != nil {
		return false, nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return false, err
	}

	userkey, err := decryptValue(key, record.EncryptedUserkey)
	if err != nil {
		log.Printf("error decrypting userkey: %s", err)
		return false, err
	}

	credentials.Username = record.Username
	credentials.Userkey = userkey
	return true, nil
}

func (credentials *Credentials) Del() error {
	db, err := getCredentialDB()
	if err != nil {
		log.Printf("error opening credential db: %s", err)
		return err
	}
	return db.Unscoped().Where("1 = 1").Delete(&credentialRecord{}).Error
}
