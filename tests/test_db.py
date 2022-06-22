import os
import pytest
import subprocess

CHECK_REF = False
BINARY = 's3-agent/s3-agent'
CONFIG_PATH = 'tests/data/'

def run_minio():
    return subprocess.run('docker run --name dev-s3 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin -p 9000:9000 -p 9001:9001 minio/minio:latest server /data --console-address ":9001"', stderr=subprocess.PIPE, stdout=subprocess.PIPE)

def stop_minio():
    return subprocess.run('docker stop dev-s3')

def run_command(cmd):
    process = subprocess.run(cmd, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    return process.returncode

def test_simple_file():
    filename = 'test.txt'
    tmp_folder_name = 'tmp'
    os.mkdir(tmp_folder_name)

    minio = run_minio()

    os.rmdir(tmp_folder_name)
