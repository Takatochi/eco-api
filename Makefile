.PHONY: demo-up demo-down run run-docker run-down fmt lint tidy test blockchain-install blockchain-deploy

demo-up:
	docker compose up -d --build

demo-down:
	docker compose down -v

run:
	./scripts/run.sh

run-docker:
	./scripts/run.sh --docker

run-down:
	./scripts/run.sh --down

fmt:
	gofmt -w .

lint:
	go vet ./...

tidy:
	go mod tidy

test:
	go test ./...

blockchain-install:
	cd blockchain && npm install

blockchain-deploy:
	cd blockchain && npx hardhat compile && npx hardhat run scripts/deploy.js --network local
