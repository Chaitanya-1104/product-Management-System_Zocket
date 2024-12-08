# product-Management-System_Zocket
## How to Run the Application

### Prerequisites
- Install [Docker](https://docs.docker.com/get-docker/)
- Install [Docker Compose](https://docs.docker.com/compose/install/)

### Steps
1. Clone the repository:<br>
   ```bash
   git clone https://github.com/Chaitanya-1104/product-Management-System_Zocket.git
2. Build and start the application using Docker Compose:<br>
   ```bash
   docker-compose up --build
3.Access the application:<br>

API: http://localhost:8080<br>
RabbitMQ Management UI: http://localhost:15672 (Default credentials: guest/guest)<br>
PostgreSQL: localhost:5432 (Use provided credentials)<br>
Redis: localhost:6379<br>
### Implementation Details:<br>
I hadn't implemented the complete features mentioned in the assignment but i tried to complete the major functionalities like:<br>
  - Created the endpoints for uploading abd retreiving products.<br>
  - Connected Successfully to postgre sql database.<br>
  - Imlpemented Redis for faster request access.<br>
  - consumer.go file is responsible for rabbitmq image processing part.main.go will call the consumer function which starts the <br>rabbitmq image 
    processing implementation.
  - I had tested the rabbit mq processing in rabbitmq web ui on the localhost its working fine i even given real image as input its working.<br>
  - I used local storage to store the compressed images due to unavailability of AWS.
  - Things which are left are handling compressed image data and added them to database and logging part.
##Thank you
