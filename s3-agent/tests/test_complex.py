import os
import pytest
import time

from .utils import assert_rclone_file, assert_entry_state, start_agent, stop_agent, FILESYSTEM_PATH


@pytest.mark.usefixtures('handle_server')
class TestS3AgentClassComplex:


    def test_import(self):
        ### GIVEN ###
        file_path = 'folder/file.txt'
        os.makedirs(f'{FILESYSTEM_PATH}/folder')
        with open(f'{FILESYSTEM_PATH}/{file_path}', 'w') as file:
            file.write('Hello world')

        ### WHEN ###
        process, connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(10)

        ### THEN ###
        assert_rclone_file(connection.cursor(), file_path)
        assert_entry_state(connection.cursor(), file_path, 11, 0, 'remote')

        with open(f'{FILESYSTEM_PATH}/{file_path}') as file:
            assert file.readlines()[0] == 'Hello world'

        assert_entry_state(connection.cursor(), file_path, 11, 1, '')

        # Reset testing environment
        stop_agent(process, connection)
