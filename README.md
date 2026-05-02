# Project Summary

This project aims to implement a face recognition system using Go and MySQL database. The system should be able to detect and classify faces in source of images folder, and classify them in folder images based on their subject names. The main features include the following:
1. Convert image to smaller size.
2. Local Face recognition to train face classification
3. Store image based on subject name

# Tech
- Go 1.25.0
    - Benefit with channel work pool to cut down the I/O process 60%
- MySQL 8.0.32
- Local compreface service

# Plans
- Local AI training service to replace compreface
- Emplement gRPC protobuf

# Project Structure
```
├── convert
│   └── image_service.go
├── face-recognition
│   ├── api
│   │   └── v1
│   ├── repo
│   ├── test
│   ├── config.go
│   ├── database.go
│   ├── repo_test.go
│   └── schema.go
├── test
│   ├── service_integration_test.go
│   └── service_test.go
├── model.go
├── repository.go
├── service.go
└──main.go
```
