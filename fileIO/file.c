#include<stdio.h>
#include<stdlib.h>

struct bio{
    char name[20];
    int age;
    float height;
};

void main(){
    FILE *fptr;
    fptr = fopen("./vijay.txt","w");
    if (fptr==NULL){
        printf("the file (vijay.txt) is not found");
        exit(3);
    }
    // writing to the file
    // fprintf(fptr,"vijay suriya");
    fprintf(fptr,"hbd vijay suriya shanmugam!");
    fclose(fptr);

    // reading from a file
    fptr = fopen("./vijay.txt","r");
    if (fptr==NULL){
        printf("the file (vijay.txt) is not found");
        exit(3);
    }
    char sent[28];
    fread(&sent, 28,1, fptr);
    printf("%s\n",sent);
    fclose(fptr);

    // seeking a file
    fptr = fopen("./vijay.txt","r");
    if (fptr==NULL){
        printf("the file (vijay.txt) is not found");
        exit(3);
    }
    fseek(fptr, 11, SEEK_SET);
    fread(&sent, 28,1, fptr);
    printf("after seeking 10 offset: %s\n",sent);
    fclose(fptr);

    fptr = fopen("./struct.bin","w+");
    if (fptr==NULL){
        printf("the file (struct.bin) is not found");
        exit(3);
    }
    struct bio vijay = {"vijay", 24, 6};
    struct bio suriya = {"suriya", 24, 6};
    fwrite(&vijay, sizeof(vijay), 1, fptr);
    fwrite(&suriya, sizeof(suriya), 1, fptr);
    fclose(fptr);

    fptr = fopen("./struct.bin","r");
    if (fptr==NULL){
        printf("the file (struct.bin) is not found");
        exit(3);
    }
    fseek(fptr,-1*28,SEEK_END);
    struct bio vijay2;
    fread(&vijay2, sizeof(vijay2), 1, fptr);
    printf("name: %s, age: %d, height: %f\n",vijay2.name, vijay2.age, vijay2.height);

    printf("size of bio struct = %d\n",sizeof(vijay));
    fclose(fptr);
}