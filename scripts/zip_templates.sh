#!/bin/bash

# Check if a path argument is provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <path_to_templates> <path_to_output>"
    exit 1
fi

# The parent directory where templates are located
parent_dir="$(pwd)/$1"
# The output directory where zip files will be placed
output_dir="$(pwd)/$2"

# Loop through each top-level template directory
for template_path in "$parent_dir"/*/; do
    # Check if it's a directory
    if [ -d "$template_path" ]; then
        # Get the name of the template directory
        template_name=$(basename "$template_path")
        # Loop through each subdirectory in the template
        for dir in "$template_path"/*; do
            # Check if it's a directory
            if [ -d "$dir" ]; then
                # Get the name of the directory
                dir_name=$(basename "$dir")
                # Create the zip file with the desired name format
                # Do so within the directory to avoid including additional parent directories
                cd $dir && zip -r "$output_dir"/"$template_name"_"$dir_name".zip .
            fi
        done
    fi
done
