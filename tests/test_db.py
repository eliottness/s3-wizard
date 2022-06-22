import os
import pytest
import sqlite3
import subprocess

CHECK_REF = False
BINARY = 's3-agent/s3-agent'
CONFIG_PATH = 'tests/data/'

def run_minio():
    return subprocess.run('docker run --name dev-s3 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin -p 9000:9000 -p 9001:9001 minio/minio:latest server --console-address ":9001"', stderr=subprocess.PIPE, stdout=subprocess.PIPE)

def stop_minio():
    return subprocess.run('docker stop dev-s3')

def run_command(cmd):
    process = subprocess.run(cmd, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    return process

def test_simple_file():
    filename = 'test.txt'
    config_name = 'simpleconfig.json'
    tmp_folder_name = 'tmp'
    db_path = 'db/'

    print(os.system('ls'))

    run_minio()

    # Starting the test
    os.mkdir(tmp_folder_name)
    os.mknod(tmp_folder_name + filename)

    process = run_command(f'./{BINARY} --config-folder={CONFIG_PATH} sync')

    # Connect to the DB to check its content
    # con = sqlite3.connect('example.db')

    os.rmdir(tmp_folder_name)
    stop_minio()
