from cProfile import run
import os
import pytest
import sqlite3
import subprocess

CHECK_REF = False
BINARY = 's3-agent/s3-agent'
CONFIG_PATH = 'tests/data/'

class TestS3AgentClass():

    def run_command(cmd, stdout=None, stderr=None, code=None):
        process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
        assert True if stdout is None else process.stdout.decode() == stdout
        assert True if stderr is None else process.stderr.decode() == stderr
        assert True if code is None else process.returncode == code

    def setup_class(self):
        cmd = 'docker run -d --name dev-s3 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin -p 9000:9000 -p 9001:9001 minio/minio:latest server /data --console-address ":9001"'
        self.run_command(cmd, code=0)

    def teardown_class(self):
        self.run_command('docker rm -f dev-s3', code=0)

    def setup_method(self):
        pass

    def teardown_method(self):
        pass

    def test_simple_file(self):
        # Todo
        pass
        # filename = 'test.txt'
        # config_name = 'simpleconfig.json'
        # tmp_folder_name = 'tmp'
        # db_path = 'db/'

        # # Starting the test
        # os.mkdir(tmp_folder_name)
        # os.mknod(tmp_folder_name + filename)

        # process = run_command(f'./{BINARY} --config-folder={CONFIG_PATH} sync')

        # # Connect to the DB to check its content
        # # con = sqlite3.connect('example.db')

        # os.rmdir(tmp_folder_name)
