import os
import pytest
import subprocess
import time


######################## UTILS ########################


NB_TRY = 5
FILESYSTEM_PATH = './tmp'


def run_command(cmd, stdout=None, stderr=None, code=None):
    process = subprocess.run(cmd.split(' '), stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    assert True if stdout is None else process.stdout.decode() == stdout, process.stdout.decode()
    assert True if stderr is None else process.stderr.decode() == stderr, process.stderr.decode()
    assert True if code is None else process.returncode == code, process.stderr.decode()


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
    # Todo: find a way to not have to fully reset env, then delete those lines
    s3_agent_path = os.path.join(os.path.expanduser('~'), '.s3-agent')
    run_command(f'rm -rf {s3_agent_path}', code=0)
    run_command(f'./s3-agent config import {request.param}', code=0)

    # Run s3-agent in sync mode
    # Todo: Pass config through CLI option once it works
    process = subprocess.Popen('./s3-agent sync'.split(' '))

    # Wait for our the filesystem to be ready
    nb_try = 0
    while not os.path.exists(FILESYSTEM_PATH) and nb_try < NB_TRY:
        time.sleep(0.1)
        nb_try += 1
    assert os.path.exists(FILESYSTEM_PATH), 'FS could not be mounted by s3-agent'

    yield

    ### TEARDOWN ###
    process.send_signal(subprocess.signal.SIGTERM)
    run_command(f'rm -rf {FILESYSTEM_PATH}', code=0)


######################## TESTS ########################


@pytest.mark.usefixtures('handle_server')
class TestS3AgentClass:

    @pytest.mark.parametrize('handle_agent', ['tests/data/simple_config.json'], indirect=True)
    def test_simple_file(self, handle_agent):
        ### GIVEN ###
        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt', 'w') as file:
            file.write('Hello world')

        ### WHEN ###
        time.sleep(5)

        ### THEN ###
        with open(f'{FILESYSTEM_PATH}/test_simple_file.txt') as file:
            assert file.readlines()[0] == 'Hello world'
