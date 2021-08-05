	mkdir ~/ccnlab/lang-acq/data/$1/$2/
	cp ~/gruntdat/wc/blanca/rohrlich/wordseg/results/active/$2/wordseg/WordSeg_Base_*tstEpcTidy.tsv ~/ccnlab/lang-acq/data/$1/$2
	cp ~/gruntdat/wc/blanca/rohrlich/wordseg/results/active/$2/wordseg/WordSeg_Base_*preTstEpcTidy.tsv ~/ccnlab/lang-acq/data/$1/$2
	cd $1/$2
    rename -s WordSeg_Base WordSeg *
    
    ../fixTstRunValues.sh
    ../fixPreTstRunValues.sh
    
    cat WordSeg_*tstEpcTidy.tsv >> WordSeg_0-24_tstEpcTidy.tsv
	cat WordSeg_*preTstEpcTidy.tsv >> WordSeg_0-24_preTstEpcTidy.tsv

	cp WordSeg_0-24_preTstEpcTidy.tsv WordSeg_0-24_prePostTstEpcTidy.tsv
	cat  WordSeg_0-24_tstEpcTidy.tsv >> WordSeg_0-24_prePostTstEpcTidy.tsv


