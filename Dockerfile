FROM debian
COPY ./bin/assignment /assignment
ENTRYPOINT /assignment