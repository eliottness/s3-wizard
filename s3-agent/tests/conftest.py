import pytest

from .utils import run_command, start_agent, stop_agent, S3_AGENT_PATH, DEBUG


@pytest.fixture(scope='class')
def handle_server():
    ### SETUP ###
    run_command('docker compose -f tests/docker-compose.yml up -d', code=0)

    yield

    ### TEARDOWN ###
    if not DEBUG:
        run_command('docker compose -f tests/docker-compose.yml down', code=0)


@pytest.fixture(scope='function')
def handle_agent(request):
    ### SETUP ###
    process, connection = start_agent(request.param)

    yield connection.cursor()

    ### TEARDOWN ###
    stop_agent(process, connection)
    run_command(f'rm -rf {S3_AGENT_PATH}', code=0)
