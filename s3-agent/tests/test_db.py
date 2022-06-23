from cProfile import run
import os
import sqlite3
import subprocess
import time

NB_TRY = 5

def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    assert True if stdout is None else process.stdout.decode() == stdout, process.stdout.decode()
    assert True if stderr is None else process.stderr.decode() == stderr, process.stderr.decode()
    assert True if code is None else process.returncode == code, process.stderr.decode()

class TestS3AgentClass:

    s3_agent_path = os.path.join(os.path.expanduser('~'), '.s3-agent')

    def setup_class(self):
        cmd = 'docker run -d --name dev-s3 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin -p 9000:9000 -p 9001:9001 minio/minio:latest server /data --console-address ":9001"'
        run_command(cmd, code=0)

    def teardown_class(self):
        run_command('docker rm -f dev-s3', code=0)

    def setup_method(self):
        # Todo: find a way to not have to fully reset env, then delete those lines
        run_command(f'rm -rf {self.s3_agent_path}', code=0)
        run_command('./s3-agent config import config.json', code=0)

        # Run s3-agent in sync mode
        self.process = subprocess.Popen('./s3-agent sync'.split(' '))

        # Wait for our the filesystem to be ready
        nb_try = 0
        while not os.path.exists('./hello') and nb_try < NB_TRY:
            time.sleep(0.1)
            nb_try += 1
        assert os.path.exists('./hello'), 'FS could not be mounted by s3-agent'

    def teardown_method(self):
        self.process.send_signal(subprocess.signal.SIGTERM)
        run_command(f'rm -rf ./hello', code=0)

    def test_simple_file(self):
        assert True
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
