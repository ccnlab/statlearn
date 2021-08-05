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
        rename -s WordSeg_Base WordSeg * ( add -n for trial run that just prints)
	

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


