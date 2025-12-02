import random

def generate_sql_file():
    filename = "populate_votersId.sql"
    total_records = 1_000_000
    batch_size = 5000  # Optimal batch size for most DBs
    
    print(f"Generating {total_records} records into {filename}...")
    
    with open(filename, 'w') as f:
        # 1. Table Schema
        f.write("DROP TABLE IF EXISTS voterId;\n")
        f.write("DROP TABLE IF EXISTS performance_log_id;\n") # Temp table for metrics
        f.write("CREATE TABLE voterId (id INT AUTO_INCREMENT PRIMARY KEY, age INT);\n")
        f.write("CREATE TABLE performance_log_id (batch_id INT, duration_ms INT);\n\n")
        
        # 2. Performance Settings
        f.write("SET autocommit=0;\n") 
        f.write("SET unique_checks=0;\n")
        f.write("SET foreign_key_checks=0;\n\n")

        batch = []
        batch_counter = 0
        
        for i in range(1, total_records + 1):
            age = random.randint(18, 90)
            batch.append(f"({age})")
            
            # When batch is full, write with timing logic
            if len(batch) == batch_size:
                batch_counter += 1
                values = ",".join(batch)
                
                # SQL Instrumentation: Measure time taken for this specific insert
                f.write(f"SET @start = NOW(6);\n")
                f.write(f"INSERT INTO voterId (age) VALUES {values};\n")
                # Log the duration in milliseconds
                f.write(f"INSERT INTO performance_log_id VALUES ({batch_counter}, TIMESTAMPDIFF(MICROSECOND, @start, NOW(6)) / 1000);\n")
                
                batch = [] 
                
        # Write remaining records
        if batch:
            batch_counter += 1
            values = ",".join(batch)
            f.write(f"SET @start = NOW(6);\n")
            f.write(f"INSERT INTO voterId (age) VALUES {values};\n")
            f.write(f"INSERT INTO performance_log_id VALUES ({batch_counter}, TIMESTAMPDIFF(MICROSECOND, @start, NOW(6)) / 1000);\n")
        
        # 3. Commit and Cleanup
        f.write("\nCOMMIT;\n")
        f.write("SET unique_checks=1;\n")
        f.write("SET foreign_key_checks=1;\n")
        
        # 4. Output the results immediately after running
        f.write("\n-- Display the performance degradation curve\n")
        f.write("SELECT * FROM performance_log_id;\n")
        
    print("Done! Import using: mysql -u root -p < populate_voters.sql")

if __name__ == "__main__":
    generate_sql_file()