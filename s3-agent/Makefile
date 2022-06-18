
RCLONE_VERSION=v1.58.1
RCLONE_ARCH=linux-amd64
RCLONE_ARCHIVE=rclone-${RCLONE_VERSION}-${RCLONE_ARCH}

all: build

rclone:
	wget "https://downloads.rclone.org/${RCLONE_VERSION}/${RCLONE_ARCHIVE}.zip"
	unzip "${RCLONE_ARCHIVE}.zip"
	cp "${RCLONE_ARCHIVE}/rclone" .
	rm -rf "${RCLONE_ARCHIVE}.zip" "${RCLONE_ARCHIVE}"

build: rclone
	go build -o s3-agent .

run: rclone
	go run .

clean:
	rm -f s3-agent rclone
