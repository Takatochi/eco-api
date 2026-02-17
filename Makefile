.PHONY: demo-up demo-down test blockchain-install blockchain-deploy

demo-up:
	docker compose up -d --build

demo-down:
	docker compose down -v

test:
	go test ./...

blockchain-install:
	cd blockchain && npm install

blockchain-deploy:
	cd blockchain && npx hardhat compile && npx hardhat run scripts/deploy.js --network local
