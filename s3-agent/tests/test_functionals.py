from importlib.resources import path
import os

import pytest
import subprocess
import time
import sqlite3


######################## UTILS ########################


NB_TRY = 10
FILESYSTEM_PATH = './tmp'
S3_AGENT_PATH = "./config"


def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    assert True if code is None else process.returncode == code, process.returncode
    assert True if stdout is None else process.stdout.decode() == stdout, process.stdout.decode()
    assert True if stderr is None else process.stderr.decode() == stderr, process.stderr.decode()


def get_rule_entry(cursor):
    cursor.execute("SELECT * FROM s3_rule_tables")
    return cursor.fetchone()


def get_node_entry(cursor, filename):
    path = os.path.join(S3_AGENT_PATH[2:], get_rule_entry(cursor)[0], filename)
    print(path)
    cursor.execute(f"SELECT * FROM s3_node_tables WHERE path = '{path}'")
    return cursor.fetchone()


def assert_rclone_file(cursor, filename):
    rule_entry = get_rule_entry(cursor)
    node_entry = get_node_entry(cursor, filename)
    s3_file_path = os.path.join("remote:bucket-test/s3-agent", rule_entry[0], node_entry[3])
    rclone_config_path = os.path.join(S3_AGENT_PATH, 'rclone.conf.tmp')
    cmd = f'./rclone --config {rclone_config_path} lsf {s3_file_path}'
    run_command(cmd, stdout=node_entry[3] + '\n', stderr='', code=0)


def assert_entry_state(cursor, filename, size, Local, server):
    entry = get_node_entry(cursor, filename)
    assert entry is not None
    assert entry[1] == size, entry
    assert entry[2] == Local, entry
    assert entry[4] == server, entry


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
    run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} config import {request.param}', code=0)
    process = subprocess.Popen(f'./s3-agent --config-folder={S3_AGENT_PATH} sync'.split(' '))

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
    run_command(f'rm -rf {S3_AGENT_PATH}', code=0)


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
        time.sleep(2)

        ### THEN ###
        assert_rclone_file(handle_agent, 'test_simple_file.txt')
        assert_entry_state(handle_agent, 'test_simple_file.txt', 11, 0, 'remote')

        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt') as file:
            assert file.readlines()[0] == 'Hello world'

        assert_entry_state(handle_agent, 'test_simple_file.txt', 11, 1, '')

    def test_simple_folder(self, handle_agent):
        ### GIVEN ###
        os.mkdir(f'{FILESYSTEM_PATH}/test_simple_folder')
        file_path = 'test_simple_folder/test_simple_file.txt'

        with open(f'{FILESYSTEM_PATH}/{file_path}', 'w') as file:
            file.write('Hello world')

        assert_entry_state(handle_agent, file_path, 0, 1, '')

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_rclone_file(handle_agent, file_path)
        assert_entry_state(handle_agent, file_path, 11, 0, 'remote')

        with open(f'{FILESYSTEM_PATH}/{file_path}') as file:
            assert file.readlines()[0] == 'Hello world'

        assert_entry_state(handle_agent, file_path, 11, 1, '')

    def test_subfolder(self, handle_agent):
        ### GIVEN ###
        os.makedirs(f'{FILESYSTEM_PATH}/folder/subfolder')
        file_path = 'folder/subfolder/test_simple_file.txt'

        with open(f'{FILESYSTEM_PATH}/{file_path}', 'w') as file:
            file.write('Hello world')

        assert_entry_state(handle_agent, file_path, 0, 1, '')

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        assert_rclone_file(handle_agent, file_path)
        assert_entry_state(handle_agent, file_path, 11, 0, 'remote')

        with open(f'{FILESYSTEM_PATH}/{file_path}') as file:
            assert file.readlines()[0] == 'Hello world'

        assert_entry_state(handle_agent, file_path, 11, 1, '')

    def test_same_name_files(self, handle_agent):
        ### GIVEN ###
        os.mkdir(f'{FILESYSTEM_PATH}/test_simple_folder')
        with open(f'{FILESYSTEM_PATH}/test_simple_folder/test_simple_file.txt', 'w') as file:
            file.write('Hello world')
        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt', 'w') as file:
            file.write('Hello world 2')

        ### WHEN ###
        time.sleep(2)

        ### THEN ###
        with open(f'{FILESYSTEM_PATH}/test_simple_folder/test_simple_file.txt') as file:
            assert file.readlines()[0] == 'Hello world'
        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt') as file:
            assert file.readlines()[0] == 'Hello world 2'
