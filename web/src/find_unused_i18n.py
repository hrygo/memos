import json
import os
import subprocess

def flatten_dict(d, parent_key='', sep='.'):
    items = []
    for k, v in d.items():
        new_key = f"{parent_key}{sep}{k}" if parent_key else k
        if isinstance(v, dict):
            items.extend(flatten_dict(v, new_key, sep=sep).items())
        else:
            items.append((new_key, v))
    return dict(items)

def main():
    en_json_path = '/Users/huangzhonghui/memos/web/src/locales/en.json'
    search_dir = '/Users/huangzhonghui/memos/web/src'
    
    if not os.path.exists(en_json_path):
        print(f"Error: {en_json_path} not found")
        return

    with open(en_json_path, 'r') as f:
        en_data = json.load(f)
    
    flat_keys = flatten_dict(en_data)
    all_keys = sorted(flat_keys.keys())
    
    print(f"Total keys: {len(all_keys)}")
    
    unused_keys = []
    for i, key in enumerate(all_keys):
        # We look for the exact string in files other than JSON files in locales
        # rg --glob '!locales/*.json' ...
        try:
            cmd = ['rg', '-l', '--glob', '!**/locales/*.json', f'["\'`]{key}["\'`]', search_dir]
            result = subprocess.run(cmd, capture_output=True, text=True)
            if result.returncode != 0:
                unused_keys.append(key)
        except Exception as e:
            print(f"Error checking {key}: {e}")
            
    print(f"Found {len(unused_keys)} potentially unused keys.")
    for key in unused_keys:
        print(key)

if __name__ == "__main__":
    main()
