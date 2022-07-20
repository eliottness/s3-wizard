import os
import pytest
import time

from .utils import create_file, assert_agent_file, start_agent, stop_agent, FILESYSTEM_PATH


@pytest.mark.usefixtures('handle_server')
class TestS3AgentClassComplex:


    def test_import_folder(self):
        ### GIVEN ###
        first_file_path = 'test_simple_file.txt'
        second_file_path = 'folder/test_simple_file.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        ### WHEN ###
        process, connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(10)

        ### THEN ###
        assert_agent_file(connection.cursor(), first_file_path, first_content)
        assert_agent_file(connection.cursor(), second_file_path, second_content)

        # Reset testing environment
        stop_agent(process, connection)


    def test_restart(self):
        ### GIVEN ###
        process, connection = start_agent('tests/data/simple_config.json')

        first_file_path = 'test_simple_file.txt'
        second_file_path = 'folder/test_simple_file.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        ### WHEN ###
        time.sleep(2)
        stop_agent(process, connection, reset_env=False)
        process, connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(2)

        ### THEN ###
        assert_agent_file(connection.cursor(), first_file_path, first_content)
        assert_agent_file(connection.cursor(), second_file_path, second_content)

        # Reset testing environment
        stop_agent(process, connection)
