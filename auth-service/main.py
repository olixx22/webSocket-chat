import grpc
from dotenv import load_dotenv
import os
import shared.gen.py.auth_pb2 as auth_pb2
import shared.gen.py.auth_pb2_grpc as auth_pb2_grpc
from shared.jwt import create_access_token, decode_token
load_dotenv()
import asyncio
from grpc.aio import server

class AuthService(auth_pb2_grpc.AuthServiceServicer):
    async def Register(self, request, context):
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



    async def Login(self, request, context):
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

    async def RefreshToken(self, request, context):
        user_id = decode_token(request.refresh_token, expected_type='refresh')

        if user_id == None:
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "Token invalid or expired")

        new_access = create_access_token(user_id, token_type='access', expires_in=1)
        new_refresh = create_access_token(user_id, token_type='refresh', expires_in=24 * 7)

        return auth_pb2.AuthResponse(
            token=new_access,
            user_id=user_id,
            refresh_token=new_refresh
        )
async def serve():

    auth_pb2_grpc.add_AuthServiceServicer_to_server(AuthService(), server())
    server.add_insecure_port('[::]:50051')
    await server().start()
    await server().wait_for_termination()


if __name__ == '__main__':
    asyncio.run(serve())