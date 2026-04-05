import grpc
from concurrent import futures
from dotenv import load_dotenv
import os
import auth_pb2
import auth_pb2_grpc
from shared.jwt import create_access_token, decode_token
load_dotenv()

class AuthService(auth_pb2_grpc.AuthServiceServicer):
    def Register(self, request, context):
        user_id = "test-uuid-12345"
        email = request.email
        password = request.password
        username = request.username
        access = create_access_token(user_id, token_type='access', expires_in=1)
        refresh = create_access_token(user_id, token_type='refresh', expires_in=24 * 7)

        return auth_pb2.AuthResponse(
            token=access,
            user_id=user_id,
            refresh_token=refresh
        )



    def Login(self, request, context):
        user_id = "test-uuid-12345"
        email = request.email
        password = request.password
        access = create_access_token(user_id, token_type='access', expires_in=1)
        refresh = create_access_token(user_id, token_type='refresh', expires_in=24 * 7)

        return auth_pb2.AuthResponse(
            token=access,
            user_id=user_id,
            refresh_token=refresh
        )

    def RefreshToken(self, request, context):
        user_id = decode_token(request.refresh_token, expected_type='refresh')

        if user_id == 'неа':
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "Token invalid or expired")

        new_access = create_access_token(user_id, token_type='access', expires_in=1)
        new_refresh = create_access_token(user_id, token_type='refresh', expires_in=24 * 7)

        return auth_pb2.AuthResponse(
            token=new_access,
            user_id=user_id,
            refresh_token=new_refresh
        )
def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    auth_pb2_grpc.add_AuthServiceServicer_to_server(AuthService(), server)

    server.add_insecure_port('[::]:50051')
    print("Сервер запущен на порту 50051...")
    server.start()
    server.wait_for_termination()


if __name__ == '__main__':
    serve()