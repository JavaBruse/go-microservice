package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go-microservice/models"

	"github.com/minio/minio-go/v7"
)

type UserService struct {
	minioClient *minio.Client
	bucketName  string
	mutex       sync.RWMutex
	nextID      int
}

func NewUserService(minioClient *minio.Client, bucketName string) *UserService {
	return &UserService{
		minioClient: minioClient,
		bucketName:  bucketName,
		nextID:      1,
	}
}

func (s *UserService) Create(user models.User) models.User {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	user.ID = s.nextID
	s.nextID++

	userData, _ := json.Marshal(user)
	objectName := fmt.Sprintf("user-%d.json", user.ID)

	_, err := s.minioClient.PutObject(
		context.Background(),
		s.bucketName,
		objectName,
		bytes.NewReader(userData),
		int64(len(userData)),
		minio.PutObjectOptions{ContentType: "application/json"},
	)

	if err != nil {
		fmt.Printf("Error saving user to MinIO: %v\n", err)
	}

	return user
}

func (s *UserService) Get(id int) (models.User, bool) {
	objectName := fmt.Sprintf("user-%d.json", id)

	obj, err := s.minioClient.GetObject(
		context.Background(),
		s.bucketName,
		objectName,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return models.User{}, false
	}
	defer obj.Close()

	var user models.User
	if err := json.NewDecoder(obj).Decode(&user); err != nil {
		return models.User{}, false
	}

	return user, true
}

func (s *UserService) GetAll() []models.User {
	var users []models.User

	objCh := s.minioClient.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{})
	for obj := range objCh {
		if obj.Err != nil {
			continue
		}

		object, err := s.minioClient.GetObject(
			context.Background(),
			s.bucketName,
			obj.Key,
			minio.GetObjectOptions{},
		)
		if err != nil {
			continue
		}

		var user models.User
		if err := json.NewDecoder(object).Decode(&user); err == nil {
			users = append(users, user)
		}
		object.Close()
	}

	return users
}

func (s *UserService) Update(user models.User) (models.User, bool) {
	if _, exists := s.Get(user.ID); !exists {
		return models.User{}, false
	}

	userData, _ := json.Marshal(user)
	objectName := fmt.Sprintf("user-%d.json", user.ID)

	_, err := s.minioClient.PutObject(
		context.Background(),
		s.bucketName,
		objectName,
		bytes.NewReader(userData),
		int64(len(userData)),
		minio.PutObjectOptions{ContentType: "application/json"},
	)

	if err != nil {
		return models.User{}, false
	}

	return user, true
}

func (s *UserService) Delete(id int) bool {
	objectName := fmt.Sprintf("user-%d.json", id)

	err := s.minioClient.RemoveObject(
		context.Background(),
		s.bucketName,
		objectName,
		minio.RemoveObjectOptions{},
	)

	return err == nil
}
