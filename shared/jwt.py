import jwt
from dotenv import load_dotenv
import os
import datetime
import uuid
load_dotenv()

SECRET_KEY = os.getenv('JWT_SECRET')
BLACKLIST = set() #Redis

def is_jti_blacklisted(jti):
    return jti in BLACKLIST

def create_access_token(user_id, token_type = 'access', expires_in=2):
    jti = str(uuid.uuid4())
    data = {
        'user_id': user_id,
        'token_type' : token_type,
        'jti' : jti,
        'exp' : datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=expires_in)

    }
    token = jwt.encode(data, SECRET_KEY, algorithm='HS256')
    return token

def decode_token (token, expected_type='access'):
    try:
        decoded = jwt.decode(token, SECRET_KEY, algorithms=['HS256'])
        jti = decoded.get('jti')
        if not jti or is_jti_blacklisted(jti):
            return 'неа'
        if decoded.get('token_type') != expected_type:return 'неа'
        user_id = decoded.get('user_id')
        return user_id
    except jwt.ExpiredSignatureError: return 'неа'
    except jwt.InvalidTokenError: return 'неа'