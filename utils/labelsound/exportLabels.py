#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""loads a wav file passed as arg1, runs audacity script to generate labels at sound start/end
(there are parameters to control the db level threshold, etc)

Based on the code from the audacity sample file pipe_test.py

Make sure Audacity is running first (and a window is open) and that mod-script-pipe is enabled (see preferences)
before running this script.

Tested on python 3.6 is strongly recommended.

"""

import os
import sys
import time
	
fn=sys.argv[1]
print (fn)

infile = "/Users/rohrlich/ccn_images/word_seg_snd_files/CVWavs_I_wInton/" + fn + ".wav"
oldname = "/Users/rohrlich/ccn_images/word_seg_snd_files/labelFiles/Label Track.txt"
newname = "/Users/rohrlich/ccn_images/word_seg_snd_files/labelFiles/" + fn + ".txt"

print("pipe-test.py, running on linux or mac")
TONAME = '/tmp/audacity_script_pipe.to.' + str(os.getuid())
FROMNAME = '/tmp/audacity_script_pipe.from.' + str(os.getuid())
EOL = '\n'

print("Write to  \"" + TONAME +"\"")
if not os.path.exists(TONAME):
    print(" ..does not exist.  Ensure Audacity is running with mod-script-pipe.")
    sys.exit()

print("Read from \"" + FROMNAME +"\"")
if not os.path.exists(FROMNAME):
    print(" ..does not exist.  Ensure Audacity is running with mod-script-pipe.")
    sys.exit()

print("-- Both pipes exist.  Good.")

TOFILE = open(TONAME, 'w')
print("-- File to write to has been opened")
FROMFILE = open(FROMNAME, 'rt')
print("-- File to read from has now been opened too\r\n")


def send_command(command):
    """Send a single command."""
    print("Senld: >>> \n"+command)
    TOFILE.write(command + EOL)
    TOFILE.flush()

def get_response():
    """Return the command response."""
    result = ''
    line = ''
    while True:
        result += line
        line = FROMFILE.readline()
        if line == '\n' and len(result) > 0:
            break
    return result

def do_command(command):
    """Send one command, and return the response."""
    send_command(command)
    response = get_response()
    print("Rcvd: <<< \n" + response)
    return response
 
def export_labels():
	print(infile)
	do_command("New:")
	do_command(f"Import2: Filename={infile}")
	do_command("SelectAll:")
	do_command("SoundFinder: sil-lev=12 sil-dur=100")
	do_command("ExportLabels:")
	time.sleep(2)
	os.rename(oldname, newname)
	
export_labels()
		