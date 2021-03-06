import os
import pytest
import sqlite3
import subprocess
import time

from .utils import assert_rclone_file, create_file, assert_agent_file, start_agent, stop_agent, run_command, get_rule_entry, assert_entry_state, S3_AGENT_PATH, FILESYSTEM_PATH


@pytest.mark.usefixtures('handle_server')
class TestS3AgentClassComplex:

    process = None
    connection = None


    def teardown_method(self, test_method):
        stop_agent(self.process, self.connection)
        self.connection = None
        self.process = None


    def test_import_simple_folder(self):
        ### GIVEN ###
        first_file_path = 'import_file_1.txt'
        second_file_path = 'folder/import_file_2.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        ### WHEN ###
        self.process, self.connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(3)

        ### THEN ###
        assert_agent_file(self.connection.cursor(), first_file_path, first_content)
        assert_agent_file(self.connection.cursor(), second_file_path, second_content)


    def test_import_deep_folder(self):
        ### GIVEN ###
        file_path = 'folder1/folder2/deep_folder_file.txt'
        content = 'Hello world'

        create_file(file_path, content)

        ### WHEN ###
        self.process, self.connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(3)

        ### THEN ###
        assert_agent_file(self.connection.cursor(), file_path, content)


    def test_restart(self):
        ### GIVEN ###
        self.process, self.connection = start_agent('tests/data/simple_config.json')

        first_file_path = 'restart_file_1.txt'
        second_file_path = 'folder/restart_file_2.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        create_file(second_file_path, second_content)

        ### WHEN ###
        time.sleep(2)
        stop_agent(self.process, self.connection, reset_env=False)
        self.process, self.connection = start_agent('tests/data/simple_config.json', reset_env=False)
        time.sleep(2)

        ### THEN ###
        assert_agent_file(self.connection.cursor(), first_file_path, first_content)
        assert_agent_file(self.connection.cursor(), second_file_path, second_content)


    def test_rebuild_mode(self):
        ### GIVEN ###
        self.process, self.connection = start_agent('tests/data/simple_config.json')

        first_file_path = 'rebuild_file_1.txt'
        second_file_path = 'folder/rebuild_file_2.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        time.sleep(2)
        create_file(second_file_path, second_content)

        ### WHEN ###
        stop_agent(self.process, reset_env=False)

        ### THEN ###
        rule_uuid = get_rule_entry(self.connection.cursor())[0]
        assert_entry_state(self.connection.cursor(), first_file_path, len(first_content), 0, 'remote')
        assert_entry_state(self.connection.cursor(), second_file_path, 0, 1, '')

        self.connection.close()

        ### WHEN ###
        run_command(f'rm {os.path.join(S3_AGENT_PATH, "sqlite.db")}', code=0)
        run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} rebuild {rule_uuid} 0', code=0)

        ### THEN ###
        self.connection = sqlite3.connect(os.path.join(S3_AGENT_PATH, 'sqlite.db'))
        assert_entry_state(self.connection.cursor(), first_file_path, len(first_content), 0, 'remote')
        assert_entry_state(self.connection.cursor(), second_file_path, len(second_content), 1, '')


    def test_dry_run_mode(self):
        ### GIVEN ###
        config_path = 'tests/data/slow_config.json'
        run_command(f'./s3-agent --config-folder={S3_AGENT_PATH} config import {config_path}', code=0)
        self.process = subprocess.Popen(f'./s3-agent --config-folder={S3_AGENT_PATH} dry-run'.split(' '))

        ### WHEN ###
        first_file_path = 'dry_run_file_1.txt'
        second_file_path = 'folder/dry_run_file_2.txt'
        first_content = 'Hello world first'
        second_content = 'Hello world second'

        create_file(first_file_path, first_content)
        time.sleep(2)
        create_file(second_file_path, second_content)
        time.sleep(2)

        ### THEN ###
        assert_rclone_file(first_file_path)
        assert_rclone_file(second_file_path, False)

        with open(f'{FILESYSTEM_PATH}/{first_file_path}', 'w') as file:
            file.write(second_content)

        time.sleep(2)

        assert_rclone_file(first_file_path, False)
        assert_rclone_file(second_file_path)
