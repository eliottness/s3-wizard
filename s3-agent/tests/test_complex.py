import os
import pytest
import sqlite3
import subprocess
import time

from .utils import assert_rclone_file, create_file, assert_agent_file, start_agent, stop_agent, run_command, get_rule_entry, assert_entry_state, S3_AGENT_PATH


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
        time.sleep(2)

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


    def test_rebuild_mode(self):
        ### GIVEN ###
        process, connection = start_agent('tests/data/simple_config.json')

        first_file_path = 'test_simple_file.txt'
        second_file_path = 'folder/test_simple_file.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        time.sleep(2)
        create_file(second_file_path, second_content)

        ### WHEN ###
        stop_agent(process, reset_env=False)

        ### THEN ###
        rule_uuid = get_rule_entry(connection.cursor())[0]
        assert_entry_state(connection.cursor(), first_file_path, len(first_content), 0, 'remote')
        assert_entry_state(connection.cursor(), second_file_path, 0, 1, '')

        connection.close()

        ### WHEN ###
        run_command(f'rm {os.path.join(S3_AGENT_PATH, "sqlite.db")}', code=0)
        run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} rebuild {rule_uuid} 0', code=0)

        ### THEN ###
        connection = sqlite3.connect(os.path.join(S3_AGENT_PATH, 'sqlite.db'))
        assert_entry_state(connection.cursor(), first_file_path, len(first_content), 0, 'remote')
        assert_entry_state(connection.cursor(), second_file_path, len(second_content), 1, '')

        # Reset testing environment
        stop_agent(connection=connection)


    def test_direct_mode(self):
        ### GIVEN ###
        config_path = 'tests/data/simple_config.json'
        run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} config import {config_path}', code=0)
        process = subprocess.Popen(f'./s3-agent --config-folder={S3_AGENT_PATH} direct'.split(' '))

        ### WHEN ###
        first_file_path = 'test_simple_file.txt'
        second_file_path = 'folder/test_simple_file.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        time.sleep(2)

        ### THEN ###
        assert_rclone_file(first_file_path)
        assert_rclone_file(second_file_path)

        stop_agent(process)
