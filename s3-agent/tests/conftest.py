import os
import pytest
import sqlite3
import subprocess
import time

from .utils import run_command


NB_TRY = 10
S3_AGENT_PATH = "./config"
FILESYSTEM_PATH = './tmp'


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
