#!/bin/bash
set -eo pipefail

usage() {
    echo "
    Usage: ./split-dir.sh [options]

    Recursively split a directory into N separate directories, maintaining
    the original file structure. Files are sorted by size and round-robbined
    into the different buckets to achieve approxiamately equal size in each
    bucket.

    Options:
        -d, --dir: Directory to split (default: ${DIR})
        -n, --num: Number of directories to split into (default: ${NUM})
        -o, --out: Output dir (default: /data/split)

        --dry-run: Sort files into buckets, but don't move them
        -h, --help: This ;P
    "
}

DIR=/opt/conda/envs/saturn
NUM=5
OUTPUT=/data/split

while [ "$1" ]; do
    case $1 in
        -d | --dir )
            shift
            DIR=$1
            ;;
        -n | --num )
            shift
            NUM=$1
            ;;
        -o | --out )
            shift
            OUTPUT=$1
            ;;
        --dry-run )
            DRY_RUN=true
            ;;
        -h | --help )
            usage
            exit 0
            ;;
        * )
            echo "Error: Unknown arg \"${1}\""
            usage
            exit 1
    esac
    shift
done

DIR=$(realpath $DIR)

file-sizes() {
    # Print <file>=<size> for every file in the directory
    local DIR=$1
    local FILES=$(ls -A $DIR | sed "s,^,$DIR/,")
    du -B 1024 -s $FILES | awk '{print $2"="$1}'
}

split() {
    # Recursively split files in the directory into approximately equal sized buckets
    local DIR=$1
    local FILE_SIZES=$(file-sizes $DIR)
    local TOTAL=$(echo -e $FILE_SIZES | sed 's/.*=//' | awk '{sum += $1} END {print sum}')
    local FILE_SIZE
    for FILE_SIZE in $FILE_SIZES; do
        # Separate <file>=<size> into seprate vars
        local FILE=${FILE_SIZE%=*}
        local SIZE=${FILE_SIZE#*=}

        # Check if new file would overload the current bucket
        if (( ${TOTALS[$CURSOR]} + $SIZE > $BUCKET_SIZE )); then
            # If file is a directory, may be able to put some of its files into the current bucket
            if [ -d $FILE ]; then
                split $FILE
                continue
            fi
            # Otherwise increment cursor to the next bucket
            CURSOR=$((($CURSOR + 1) % $NUM))
        fi
        # Store filepath in the bucket, and increase its size counter
        BUCKETS[$CURSOR]+="$FILE "
        TOTALS[$CURSOR]=$(( ${TOTALS[$CURSOR]} + $SIZE ))
    done
}

declare -A BUCKETS
declare -A TOTALS
KEYS=$(seq 0 $(( $NUM - 1)))
CURSOR=0
TOTAL_SIZE=$(file-sizes $DIR | sed 's/.*=//' | awk '{sum += $1} END {print sum}')
BUCKET_SIZE=$(( $TOTAL_SIZE / $NUM ))

# Sort files into buckets
split $DIR

# Move files to their associated bucket
for i in $KEYS; do
    SPLIT_DIR=${OUTPUT}/${i}
    BUCKET_DIR=${SPLIT_DIR}${DIR}
    echo -n "Creating $BUCKET_DIR"

    if [ "$DRY_RUN" ]; then
        echo " (dry-run)"
        echo "  Size: $(echo ${TOTALS[$i]} | awk '{print $1/1000"MB"}')"
    else
        echo
        mkdir -p $BUCKET_DIR
        for FILE in ${BUCKETS[$i]}; do
            NEW_FILE=$(echo $FILE | sed "s,^${DIR},${BUCKET_DIR},")
            mkdir -p $(dirname $NEW_FILE)
            mv $FILE $NEW_FILE
        done
    fi
done
