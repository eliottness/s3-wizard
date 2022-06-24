from importlib.resources import path
import os
import pytest
import subprocess
import time
import sqlite3


######################## UTILS ########################


NB_TRY = 5
FILESYSTEM_PATH = './tmp'
S3_AGENT_PATH = os.path.join(os.path.expanduser('~'), '.s3-agent')


def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    assert True if code is None else process.returncode == code, process.returncode
    assert True if stdout is None else process.stdout.decode() == stdout, process.stdout.decode()
    assert True if stderr is None else process.stderr.decode() == stderr, process.stderr.decode()


def assert_rclone_file(cursor, filename):
    cursor.execute("SELECT * FROM s3_node_tables WHERE path LIKE '%'||?||'%'", (filename,))
    s3_file_path = os.path.join(cursor.fetchone()[3], filename)
    rclone_config_path = os.path.join(S3_AGENT_PATH, 'rclone.conf.tmp')
    cmd = f'./rclone --config {rclone_config_path} lsf remote:bucket-test/s3-agent/{s3_file_path}'
    run_command(cmd, stdout=filename + '\n', stderr='', code=0)


def assert_entry_state(cursor, filename, size, Local, server):
    cursor.execute("SELECT * FROM s3_node_tables WHERE path LIKE '%'||?||'%'", (filename,))
    row = cursor.fetchone()
    assert row is not None
    assert row[1] == size, row
    assert row[2] == Local, row
    assert row[4] == server, row


####################### FIXTURES ######################


@pytest.fixture(scope='class')
def handle_server():
    ### SETUP ###
    run_command('docker compose -f tests/docker-compose.yml up -d', code=0)

    yield

    ### TEARDOWN ###
    run_command('docker compose -f tests/docker-compose.yml down', code=0)


@pytest.fixture(scope='function')
def handle_agent(request):
    ### SETUP ###
    # Reset env
    run_command(f'rm -rf {S3_AGENT_PATH}', code=0)

    # Set config then run s3-agent in sync mode
    run_command(f'./s3-agent config import {request.param}', code=0)
    process = subprocess.Popen('./s3-agent sync'.split(' '))

    # Wait for our the filesystem to be ready
    nb_try = 0
    while not os.path.exists(FILESYSTEM_PATH) and nb_try < NB_TRY:
        time.sleep(0.1)
        nb_try += 1
    assert os.path.exists(FILESYSTEM_PATH), 'FS could not be mounted by s3-agent'

    # Connect to our local db
    connection = sqlite3.connect(os.path.join(S3_AGENT_PATH, 'sqlite.db'))

    yield connection.cursor()

    ### TEARDOWN ###
    connection.close()
    process.send_signal(subprocess.signal.SIGTERM)
    process.wait()
    run_command(f'test -e {FILESYSTEM_PATH}', code=1)


######################## TESTS ########################


@pytest.mark.usefixtures('handle_server')
@pytest.mark.parametrize('handle_agent', ['tests/data/simple_config.json'], indirect=True)
class TestS3AgentClass:

    def test_simple_file(self, handle_agent):
        ### GIVEN ###
        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt', 'w') as file:
            file.write('Hello world')
        assert_entry_state(handle_agent, 'test_simple_file.txt', 0, 1, '')

        ### WHEN ###
        time.sleep(10)

        ### THEN ###
        assert_rclone_file(handle_agent, 'test_simple_file.txt')
        assert_entry_state(handle_agent, 'test_simple_file.txt', 0, 0, 'remote')

        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt') as file:
            assert file.readlines()[0] == 'Hello world'

        assert_entry_state(handle_agent, 'test_simple_file.txt', 0, 1, '')
