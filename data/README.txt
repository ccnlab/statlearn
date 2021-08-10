Data preparation

There is a script called dataPrep.sh that will do all of the steps. Here they are with a bit of explanation.

Just go to the data directory execute

./dataPrep.sh

---------------------------------------------

Step 1 - copy the "tidy" data from the git results repository to this directory and rename to be a bit shorter

for example:

	mkdir ~/ccnlab/lang-acq/data/Saffran/roh002264/
	cp ~/gruntdat/wc/blanca/rohrlich/wordseg/results/active/roh002264/wordseg/WordSeg_Base_*tstEpcTidy.tsv ~/ccnlab/lang-acq/data/Saffran/roh002264
	cp ~/gruntdat/wc/blanca/rohrlich/wordseg/results/active/roh002264/wordseg/WordSeg_Base_*preTstEpcTidy.tsv ~/ccnlab/lang-acq/data/Saffran/roh002264
	cd Saffran/roh002264
        rename -s WordSeg_Base WordSeg * ( add -n for trial run that just prints) (brew install rename if you don't have the utility already)
	

Step 2 - every file has the run number as zero. These need to be changed to match the actual run which is shown in the file name. Use the fixTstRunValues.sh script

for example:

	../fixTstRunValues.sh  (../ because the fix script is a directory above the actual data)
	../fixPreTstRunValues.sh

Step 3 - concatenate all of the runs into a single file, where the 0-24 indicates the run numbers, change as appropriate. Do this for both tst and preTst

	cat WordSeg_*tstEpcTidy.tsv >> WordSeg_0-24_tstEpcTidy.tsv
	cat WordSeg_*preTstEpcTidy.tsv >> WordSeg_0-24_preTstEpcTidy.tsv

Step 4 - concatenate the pretest and test data WordSeg_*tstEpcTidy.tsv 

	cp WordSeg_0-24_preTstEpcTidy.tsv WordSeg_0-24_prePostTstEpcTidy.tsv (copy pretest data)
	cat  WordSeg_0-24_tstEpcTidy.tsv >> WordSeg_0-24_prePostTstEpcTidy.tsv (add the test data)

Note
- The runs without pretraining were done after moving over to the hpc2 server and from the "statlearn" project which is the clean copy for public access.
- The runs with the pretraining were done on the boulder blanca server from the "lang-acq" project. The code is identical.
- The script is only for the runs with pretraining but the changes for processing the data from the no pretrain runs are simple and obvious
