import os
import pytest
import time

from .utils import assert_rclone_file, assert_entry_state


FILESYSTEM_PATH = './tmp'


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
