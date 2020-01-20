FROM python:3

WORKDIR /app
COPY . .
RUN pip install -r requirements.txt

EXPOSE 8888

CMD ["python3", "auth_server.py"]
