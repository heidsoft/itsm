# ITSM 系统 AI 服务集成架构

## 1. AI 服务架构概览

### 1.1 整体架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    ITSM 前端应用                              │
├─────────────────────────────────────────────────────────────┤
│                    ITSM 后端服务                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   工单服务   │  │   事件服务   │  │   变更服务   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                    AI 服务层                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │  智能分析    │  │  预测维护    │  │  知识推荐    │          │
│  │   服务      │  │   服务      │  │   服务      │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│                    AI 基础设施层                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   LLM 服务   │  │  向量数据库  │  │  机器学习    │          │
│  │  (OpenAI)   │  │ (Qdrant)    │  │  模型服务    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 技术栈选型

#### AI/ML 框架
- **LLM**: OpenAI GPT-4, Claude, 或本地部署的开源模型
- **向量数据库**: Qdrant 或 Pinecone
- **机器学习**: Python + scikit-learn + TensorFlow/PyTorch
- **自然语言处理**: spaCy, NLTK, Transformers
- **时间序列分析**: Prophet, ARIMA

#### 基础设施
- **消息队列**: Redis/RabbitMQ
- **缓存**: Redis
- **任务调度**: Celery
- **监控**: Prometheus + Grafana

## 2. 智能分析服务

### 2.1 工单智能分类

```python
# ai_services/ticket_classifier.py
from typing import Dict, List, Optional
import openai
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.naive_bayes import MultinomialNB
import joblib

class TicketClassifier:
    def __init__(self):
        self.llm_client = openai.OpenAI()
        self.traditional_model = None
        self.vectorizer = None
        self.load_models()
    
    def load_models(self):
        """加载预训练的传统ML模型"""
        try:
            self.traditional_model = joblib.load('models/ticket_classifier.pkl')
            self.vectorizer = joblib.load('models/tfidf_vectorizer.pkl')
        except FileNotFoundError:
            self.train_traditional_model()
    
    async def classify_ticket(self, title: str, description: str) -> Dict:
        """智能分类工单"""
        # 使用LLM进行分类
        llm_result = await self.classify_with_llm(title, description)
        
        # 使用传统ML模型进行分类
        traditional_result = self.classify_with_traditional_ml(title, description)
        
        # 融合结果
        final_result = self.ensemble_results(llm_result, traditional_result)
        
        return {
            'category': final_result['category'],
            'priority': final_result['priority'],
            'urgency': final_result['urgency'],
            'impact': final_result['impact'],
            'confidence': final_result['confidence'],
            'suggested_assignee': final_result.get('suggested_assignee'),
            'reasoning': final_result.get('reasoning')
        }
    
    async def classify_with_llm(self, title: str, description: str) -> Dict:
        """使用LLM进行分类"""
        prompt = f"""
        作为ITSM系统的智能分析专家，请分析以下工单并提供分类建议：
        
        标题: {title}
        描述: {description}
        
        请提供以下信息：
        1. 类别 (硬件问题、软件问题、网络问题、权限问题、其他)
        2. 优先级 (低、中、高、紧急)
        3. 紧急程度 (低、中、高)
        4. 影响范围 (低、中、高)
        5. 建议处理人员类型
        6. 分析理由
        
        请以JSON格式返回结果。
        """
        
        response = await self.llm_client.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": prompt}],
            temperature=0.1
        )
        
        # 解析LLM响应
        return self.parse_llm_response(response.choices[0].message.content)
    
    def classify_with_traditional_ml(self, title: str, description: str) -> Dict:
        """使用传统ML模型进行分类"""
        if not self.traditional_model:
            return {}
        
        text = f"{title} {description}"
        features = self.vectorizer.transform([text])
        
        category_prob = self.traditional_model.predict_proba(features)[0]
        category = self.traditional_model.classes_[category_prob.argmax()]
        confidence = category_prob.max()
        
        return {
            'category': category,
            'confidence': confidence
        }
    
    def ensemble_results(self, llm_result: Dict, traditional_result: Dict) -> Dict:
        """融合LLM和传统ML的结果"""
        # 实现结果融合逻辑
        if llm_result.get('confidence', 0) > 0.8:
            return llm_result
        elif traditional_result.get('confidence', 0) > 0.7:
            # 结合两种方法的结果
            return {
                **llm_result,
                'category': traditional_result['category'],
                'confidence': (llm_result.get('confidence', 0) + traditional_result.get('confidence', 0)) / 2
            }
        else:
            return llm_result
```

### 2.2 相似工单检索

```python
# ai_services/similarity_search.py
from typing import List, Dict
import numpy as np
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct
from sentence_transformers import SentenceTransformer

class SimilaritySearchService:
    def __init__(self):
        self.qdrant_client = QdrantClient(host="localhost", port=6333)
        self.encoder = SentenceTransformer('all-MiniLM-L6-v2')
        self.collection_name = "tickets"
        self.setup_collection()
    
    def setup_collection(self):
        """初始化向量数据库集合"""
        try:
            self.qdrant_client.create_collection(
                collection_name=self.collection_name,
                vectors_config=VectorParams(size=384, distance=Distance.COSINE)
            )
        except Exception:
            pass  # 集合已存在
    
    async def index_ticket(self, ticket_id: str, title: str, description: str, 
                          category: str, resolution: str = None):
        """将工单索引到向量数据库"""
        text = f"{title} {description}"
        if resolution:
            text += f" {resolution}"
        
        vector = self.encoder.encode(text).tolist()
        
        point = PointStruct(
            id=ticket_id,
            vector=vector,
            payload={
                "title": title,
                "description": description,
                "category": category,
                "resolution": resolution
            }
        )
        
        self.qdrant_client.upsert(
            collection_name=self.collection_name,
            points=[point]
        )
    
    async def find_similar_tickets(self, title: str, description: str, 
                                 limit: int = 5) -> List[Dict]:
        """查找相似工单"""
        query_text = f"{title} {description}"
        query_vector = self.encoder.encode(query_text).tolist()
        
        search_result = self.qdrant_client.search(
            collection_name=self.collection_name,
            query_vector=query_vector,
            limit=limit,
            score_threshold=0.7
        )
        
        similar_tickets = []
        for result in search_result:
            similar_tickets.append({
                "id": result.id,
                "title": result.payload["title"],
                "description": result.payload["description"],
                "category": result.payload["category"],
                "resolution": result.payload.get("resolution"),
                "similarity": result.score
            })
        
        return similar_tickets
    
    async def suggest_resolution(self, ticket_id: str, title: str, 
                               description: str) -> Dict:
        """基于相似工单建议解决方案"""
        similar_tickets = await self.find_similar_tickets(title, description)
        
        if not similar_tickets:
            return {"suggestions": [], "confidence": 0}
        
        # 提取解决方案
        resolutions = [t["resolution"] for t in similar_tickets 
                      if t["resolution"]]
        
        if not resolutions:
            return {"suggestions": [], "confidence": 0}
        
        # 使用LLM生成综合建议
        suggestions = await self.generate_resolution_suggestions(
            title, description, resolutions
        )
        
        return {
            "suggestions": suggestions,
            "similar_tickets": similar_tickets[:3],
            "confidence": similar_tickets[0]["similarity"] if similar_tickets else 0
        }
```

### 2.3 情感分析服务

```python
# ai_services/sentiment_analysis.py
from typing import Dict
import openai
from transformers import pipeline

class SentimentAnalysisService:
    def __init__(self):
        self.llm_client = openai.OpenAI()
        # 使用预训练的情感分析模型
        self.sentiment_pipeline = pipeline(
            "sentiment-analysis",
            model="cardiffnlp/twitter-roberta-base-sentiment-latest"
        )
    
    async def analyze_sentiment(self, text: str) -> Dict:
        """分析文本情感"""
        # 使用Transformers模型进行基础情感分析
        basic_result = self.sentiment_pipeline(text)[0]
        
        # 使用LLM进行详细分析
        detailed_result = await self.analyze_with_llm(text)
        
        return {
            "sentiment": self.map_sentiment(basic_result["label"]),
            "confidence": basic_result["score"],
            "urgency_indicators": detailed_result.get("urgency_indicators", []),
            "emotional_state": detailed_result.get("emotional_state"),
            "priority_suggestion": detailed_result.get("priority_suggestion")
        }
    
    async def analyze_with_llm(self, text: str) -> Dict:
        """使用LLM进行详细情感分析"""
        prompt = f"""
        分析以下工单文本的情感和紧急程度：
        
        文本: {text}
        
        请提供：
        1. 用户的情绪状态 (平静、焦虑、愤怒、沮丧等)
        2. 紧急程度指标 (如"立即"、"紧急"、"ASAP"等关键词)
        3. 建议的优先级调整
        4. 处理建议
        
        以JSON格式返回。
        """
        
        response = await self.llm_client.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": prompt}],
            temperature=0.1
        )
        
        return self.parse_llm_response(response.choices[0].message.content)
    
    def map_sentiment(self, label: str) -> str:
        """映射情感标签"""
        mapping = {
            "LABEL_0": "negative",
            "LABEL_1": "neutral", 
            "LABEL_2": "positive",
            "NEGATIVE": "negative",
            "NEUTRAL": "neutral",
            "POSITIVE": "positive"
        }
        return mapping.get(label, "neutral")
```

## 3. 预测性维护服务

### 3.1 故障预测模型

```python
# ai_services/predictive_maintenance.py
import pandas as pd
import numpy as np
from sklearn.ensemble import IsolationForest, RandomForestRegressor
from sklearn.preprocessing import StandardScaler
from prophet import Prophet
import joblib
from datetime import datetime, timedelta

class PredictiveMaintenanceService:
    def __init__(self):
        self.anomaly_detector = IsolationForest(contamination=0.1)
        self.failure_predictor = RandomForestRegressor(n_estimators=100)
        self.scaler = StandardScaler()
        self.load_models()
    
    def load_models(self):
        """加载预训练模型"""
        try:
            self.anomaly_detector = joblib.load('models/anomaly_detector.pkl')
            self.failure_predictor = joblib.load('models/failure_predictor.pkl')
            self.scaler = joblib.load('models/scaler.pkl')
        except FileNotFoundError:
            pass  # 模型不存在，需要训练
    
    async def predict_failures(self, service_id: str, 
                             historical_data: List[Dict]) -> Dict:
        """预测服务故障"""
        if not historical_data:
            return {"predictions": [], "confidence": 0}
        
        # 准备数据
        df = pd.DataFrame(historical_data)
        features = self.extract_features(df)
        
        # 异常检测
        anomalies = self.detect_anomalies(features)
        
        # 故障时间预测
        failure_predictions = self.predict_failure_time(df)
        
        # 风险评估
        risk_assessment = self.assess_risk(features, anomalies)
        
        return {
            "service_id": service_id,
            "predictions": failure_predictions,
            "anomalies": anomalies,
            "risk_level": risk_assessment["level"],
            "risk_factors": risk_assessment["factors"],
            "recommendations": self.generate_recommendations(risk_assessment),
            "confidence": risk_assessment["confidence"]
        }
    
    def extract_features(self, df: pd.DataFrame) -> np.ndarray:
        """提取特征"""
        features = []
        
        # 时间特征
        df['hour'] = pd.to_datetime(df['timestamp']).dt.hour
        df['day_of_week'] = pd.to_datetime(df['timestamp']).dt.dayofweek
        
        # 统计特征
        numeric_columns = df.select_dtypes(include=[np.number]).columns
        for col in numeric_columns:
            if col not in ['hour', 'day_of_week']:
                features.extend([
                    df[col].mean(),
                    df[col].std(),
                    df[col].max(),
                    df[col].min(),
                    df[col].quantile(0.95)
                ])
        
        # 趋势特征
        for col in numeric_columns:
            if col not in ['hour', 'day_of_week'] and len(df) > 1:
                trend = np.polyfit(range(len(df)), df[col], 1)[0]
                features.append(trend)
        
        return np.array(features).reshape(1, -1)
    
    def detect_anomalies(self, features: np.ndarray) -> List[Dict]:
        """检测异常"""
        scaled_features = self.scaler.transform(features)
        anomaly_score = self.anomaly_detector.decision_function(scaled_features)[0]
        is_anomaly = self.anomaly_detector.predict(scaled_features)[0] == -1
        
        return [{
            "is_anomaly": bool(is_anomaly),
            "anomaly_score": float(anomaly_score),
            "timestamp": datetime.now().isoformat()
        }]
    
    def predict_failure_time(self, df: pd.DataFrame) -> List[Dict]:
        """预测故障时间"""
        predictions = []
        
        # 使用Prophet进行时间序列预测
        for metric in ['cpu_usage', 'memory_usage', 'disk_usage', 'error_rate']:
            if metric in df.columns:
                prophet_df = df[['timestamp', metric]].rename(
                    columns={'timestamp': 'ds', metric: 'y'}
                )
                
                model = Prophet(
                    changepoint_prior_scale=0.05,
                    seasonality_prior_scale=10,
                    holidays_prior_scale=10,
                    daily_seasonality=True,
                    weekly_seasonality=True,
                    yearly_seasonality=False
                )
                
                model.fit(prophet_df)
                
                # 预测未来7天
                future = model.make_future_dataframe(periods=7, freq='D')
                forecast = model.predict(future)
                
                # 检查是否有超过阈值的预测
                threshold = self.get_threshold(metric)
                future_issues = forecast[forecast['yhat'] > threshold]
                
                if not future_issues.empty:
                    predictions.append({
                        "metric": metric,
                        "predicted_failure_date": future_issues.iloc[0]['ds'].isoformat(),
                        "predicted_value": float(future_issues.iloc[0]['yhat']),
                        "threshold": threshold,
                        "confidence": float(1 - future_issues.iloc[0]['yhat_lower'] / future_issues.iloc[0]['yhat_upper'])
                    })
        
        return predictions
    
    def assess_risk(self, features: np.ndarray, anomalies: List[Dict]) -> Dict:
        """评估风险"""
        risk_score = 0
        risk_factors = []
        
        # 异常检测结果
        if anomalies and anomalies[0]["is_anomaly"]:
            risk_score += 0.4
            risk_factors.append("检测到异常行为模式")
        
        # 特征分析
        if len(features[0]) > 0:
            # 假设特征已经标准化
            high_values = np.sum(np.abs(features[0]) > 2)
            if high_values > len(features[0]) * 0.3:
                risk_score += 0.3
                risk_factors.append("多个指标异常")
        
        # 确定风险等级
        if risk_score >= 0.7:
            level = "high"
        elif risk_score >= 0.4:
            level = "medium"
        else:
            level = "low"
        
        return {
            "level": level,
            "score": risk_score,
            "factors": risk_factors,
            "confidence": min(risk_score + 0.2, 1.0)
        }
    
    def generate_recommendations(self, risk_assessment: Dict) -> List[str]:
        """生成维护建议"""
        recommendations = []
        
        if risk_assessment["level"] == "high":
            recommendations.extend([
                "立即检查系统状态",
                "准备故障转移方案",
                "通知相关技术团队",
                "安排紧急维护窗口"
            ])
        elif risk_assessment["level"] == "medium":
            recommendations.extend([
                "增加监控频率",
                "安排预防性维护",
                "检查相关配置项",
                "准备备用资源"
            ])
        else:
            recommendations.extend([
                "继续常规监控",
                "定期健康检查",
                "保持当前维护计划"
            ])
        
        return recommendations
    
    def get_threshold(self, metric: str) -> float:
        """获取指标阈值"""
        thresholds = {
            "cpu_usage": 85.0,
            "memory_usage": 90.0,
            "disk_usage": 85.0,
            "error_rate": 5.0
        }
        return thresholds.get(metric, 80.0)
```

### 3.2 容量规划服务

```python
# ai_services/capacity_planning.py
from typing import Dict, List
import pandas as pd
import numpy as np
from prophet import Prophet
from sklearn.linear_model import LinearRegression

class CapacityPlanningService:
    def __init__(self):
        self.growth_models = {}
    
    async def analyze_capacity_trends(self, service_id: str, 
                                    metrics_data: List[Dict],
                                    forecast_days: int = 90) -> Dict:
        """分析容量趋势"""
        df = pd.DataFrame(metrics_data)
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        
        results = {}
        
        for metric in ['cpu_usage', 'memory_usage', 'disk_usage', 'network_io']:
            if metric in df.columns:
                forecast = self.forecast_metric(df, metric, forecast_days)
                capacity_analysis = self.analyze_capacity_requirements(forecast, metric)
                
                results[metric] = {
                    "current_usage": float(df[metric].iloc[-1]),
                    "trend": self.calculate_trend(df[metric]),
                    "forecast": forecast,
                    "capacity_analysis": capacity_analysis,
                    "recommendations": self.generate_capacity_recommendations(
                        capacity_analysis, metric
                    )
                }
        
        return {
            "service_id": service_id,
            "analysis_date": datetime.now().isoformat(),
            "forecast_period_days": forecast_days,
            "metrics": results,
            "overall_recommendation": self.generate_overall_recommendation(results)
        }
    
    def forecast_metric(self, df: pd.DataFrame, metric: str, 
                       forecast_days: int) -> Dict:
        """预测指标趋势"""
        prophet_df = df[['timestamp', metric]].rename(
            columns={'timestamp': 'ds', metric: 'y'}
        )
        
        model = Prophet(
            changepoint_prior_scale=0.05,
            seasonality_prior_scale=10,
            daily_seasonality=True,
            weekly_seasonality=True,
            yearly_seasonality=False
        )
        
        model.fit(prophet_df)
        
        future = model.make_future_dataframe(periods=forecast_days, freq='D')
        forecast = model.predict(future)
        
        # 提取预测结果
        future_forecast = forecast.tail(forecast_days)
        
        return {
            "dates": future_forecast['ds'].dt.strftime('%Y-%m-%d').tolist(),
            "predicted_values": future_forecast['yhat'].tolist(),
            "lower_bound": future_forecast['yhat_lower'].tolist(),
            "upper_bound": future_forecast['yhat_upper'].tolist(),
            "trend": future_forecast['trend'].tolist()
        }
    
    def analyze_capacity_requirements(self, forecast: Dict, metric: str) -> Dict:
        """分析容量需求"""
        predicted_values = forecast["predicted_values"]
        upper_bound = forecast["upper_bound"]
        
        # 获取阈值
        threshold = self.get_capacity_threshold(metric)
        
        # 找到超过阈值的时间点
        breach_points = []
        for i, (pred, upper) in enumerate(zip(predicted_values, upper_bound)):
            if upper > threshold:
                breach_points.append({
                    "date": forecast["dates"][i],
                    "predicted_value": pred,
                    "upper_bound": upper,
                    "days_from_now": i + 1
                })
        
        # 计算增长率
        if len(predicted_values) > 1:
            growth_rate = (predicted_values[-1] - predicted_values[0]) / len(predicted_values)
        else:
            growth_rate = 0
        
        return {
            "threshold": threshold,
            "max_predicted_value": max(predicted_values),
            "growth_rate_per_day": growth_rate,
            "breach_points": breach_points,
            "time_to_capacity_limit": breach_points[0]["days_from_now"] if breach_points else None,
            "capacity_utilization_forecast": max(predicted_values) / 100.0  # 假设100%为最大容量
        }
    
    def generate_capacity_recommendations(self, analysis: Dict, metric: str) -> List[str]:
        """生成容量建议"""
        recommendations = []
        
        if analysis["breach_points"]:
            days_to_breach = analysis["breach_points"][0]["days_from_now"]
            
            if days_to_breach <= 30:
                recommendations.append(f"紧急：{metric}将在{days_to_breach}天内达到容量限制")
                recommendations.append("立即安排容量扩展")
                recommendations.append("考虑负载均衡或优化")
            elif days_to_breach <= 60:
                recommendations.append(f"警告：{metric}将在{days_to_breach}天内达到容量限制")
                recommendations.append("开始规划容量扩展")
                recommendations.append("监控使用趋势变化")
            else:
                recommendations.append(f"注意：{metric}预计在{days_to_breach}天后达到容量限制")
                recommendations.append("将容量扩展纳入长期规划")
        
        # 基于增长率的建议
        if analysis["growth_rate_per_day"] > 1:
            recommendations.append("检测到快速增长趋势，建议增加监控频率")
        elif analysis["growth_rate_per_day"] < 0:
            recommendations.append("使用量呈下降趋势，可考虑资源优化")
        
        return recommendations
    
    def get_capacity_threshold(self, metric: str) -> float:
        """获取容量阈值"""
        thresholds = {
            "cpu_usage": 80.0,
            "memory_usage": 85.0,
            "disk_usage": 80.0,
            "network_io": 75.0
        }
        return thresholds.get(metric, 80.0)
```

## 4. 知识推荐服务

### 4.1 智能知识库

```python
# ai_services/knowledge_recommendation.py
from typing import List, Dict
import openai
from qdrant_client import QdrantClient
from sentence_transformers import SentenceTransformer

class KnowledgeRecommendationService:
    def __init__(self):
        self.llm_client = openai.OpenAI()
        self.qdrant_client = QdrantClient(host="localhost", port=6333)
        self.encoder = SentenceTransformer('all-MiniLM-L6-v2')
        self.kb_collection = "knowledge_base"
        self.setup_knowledge_base()
    
    def setup_knowledge_base(self):
        """初始化知识库"""
        try:
            self.qdrant_client.create_collection(
                collection_name=self.kb_collection,
                vectors_config=VectorParams(size=384, distance=Distance.COSINE)
            )
        except Exception:
            pass
    
    async def recommend_solutions(self, ticket_id: str, title: str, 
                                description: str, category: str) -> Dict:
        """推荐解决方案"""
        # 搜索相关知识库文章
        kb_articles = await self.search_knowledge_base(title, description, category)
        
        # 搜索相似的已解决工单
        similar_tickets = await self.search_resolved_tickets(title, description)
        
        # 使用LLM生成综合建议
        ai_suggestions = await self.generate_ai_suggestions(
            title, description, kb_articles, similar_tickets
        )
        
        return {
            "ticket_id": ticket_id,
            "knowledge_base_articles": kb_articles,
            "similar_resolved_tickets": similar_tickets,
            "ai_generated_suggestions": ai_suggestions,
            "confidence_score": self.calculate_confidence_score(
                kb_articles, similar_tickets, ai_suggestions
            )
        }
    
    async def search_knowledge_base(self, title: str, description: str, 
                                  category: str) -> List[Dict]:
        """搜索知识库"""
        query_text = f"{title} {description} {category}"
        query_vector = self.encoder.encode(query_text).tolist()
        
        search_result = self.qdrant_client.search(
            collection_name=self.kb_collection,
            query_vector=query_vector,
            limit=5,
            score_threshold=0.6
        )
        
        articles = []
        for result in search_result:
            articles.append({
                "id": result.id,
                "title": result.payload["title"],
                "summary": result.payload["summary"],
                "content": result.payload["content"],
                "category": result.payload["category"],
                "tags": result.payload.get("tags", []),
                "relevance_score": result.score,
                "last_updated": result.payload.get("last_updated")
            })
        
        return articles
    
    async def generate_ai_suggestions(self, title: str, description: str,
                                    kb_articles: List[Dict],
                                    similar_tickets: List[Dict]) -> List[Dict]:
        """生成AI建议"""
        context = self.build_context(kb_articles, similar_tickets)
        
        prompt = f"""
        基于以下工单信息和相关资料，提供详细的解决建议：
        
        工单标题: {title}
        工单描述: {description}
        
        相关知识库文章:
        {self.format_kb_articles(kb_articles)}
        
        相似已解决工单:
        {self.format_similar_tickets(similar_tickets)}
        
        请提供：
        1. 3-5个具体的解决步骤
        2. 每个步骤的详细说明
        3. 可能的风险和注意事项
        4. 预估解决时间
        5. 需要的技能或权限
        
        以JSON格式返回建议列表。
        """
        
        response = await self.llm_client.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": prompt}],
            temperature=0.2
        )
        
        return self.parse_ai_suggestions(response.choices[0].message.content)
    
    async def update_knowledge_base(self, ticket_id: str, title: str,
                                  description: str, resolution: str,
                                  category: str, tags: List[str] = None):
        """更新知识库"""
        # 检查是否应该创建新的知识库条目
        should_create = await self.should_create_kb_entry(
            title, description, resolution
        )
        
        if should_create:
            # 生成知识库文章
            article = await self.generate_kb_article(
                title, description, resolution, category, tags or []
            )
            
            # 索引到向量数据库
            await self.index_kb_article(article)
            
            return article
        
        return None
    
    async def should_create_kb_entry(self, title: str, description: str,
                                   resolution: str) -> bool:
        """判断是否应该创建知识库条目"""
        # 检查是否已有相似的知识库文章
        existing_articles = await self.search_knowledge_base(title, description, "")
        
        if existing_articles and existing_articles[0]["relevance_score"] > 0.9:
            return False
        
        # 使用LLM判断价值
        prompt = f"""
        判断以下工单解决方案是否有价值创建知识库文章：
        
        标题: {title}
        描述: {description}
        解决方案: {resolution}
        
        考虑因素：
        1. 问题的通用性
        2. 解决方案的复杂性
        3. 可能的复用价值
        4. 技术深度
        
        返回 true 或 false。
        """
        
        response = await self.llm_client.chat.completions.create(
            model="gpt-4",
            messages=[{"role": "user", "content": prompt}],
            temperature=0.1
        )
        
        return "true" in response.choices[0].message.content.lower()
```

## 5. AI 服务集成层

### 5.1 AI 服务协调器

```python
# ai_services/ai_coordinator.py
from typing import Dict, List, Optional
import asyncio
from dataclasses import dataclass
from enum import Enum

class AIServiceType(Enum):
    TICKET_CLASSIFICATION = "ticket_classification"
    SIMILARITY_SEARCH = "similarity_search"
    SENTIMENT_ANALYSIS = "sentiment_analysis"
    PREDICTIVE_MAINTENANCE = "predictive_maintenance"
    KNOWLEDGE_RECOMMENDATION = "knowledge_recommendation"

@dataclass
class AIRequest:
    service_type: AIServiceType
    data: Dict
    priority: int = 1
    timeout: int = 30

class AICoordinator:
    def __init__(self):
        self.services = {
            AIServiceType.TICKET_CLASSIFICATION: TicketClassifier(),
            AIServiceType.SIMILARITY_SEARCH: SimilaritySearchService(),
            AIServiceType.SENTIMENT_ANALYSIS: SentimentAnalysisService(),
            AIServiceType.PREDICTIVE_MAINTENANCE: PredictiveMaintenanceService(),
            AIServiceType.KNOWLEDGE_RECOMMENDATION: KnowledgeRecommendationService()
        }
        self.request_queue = asyncio.Queue()
        self.results_cache = {}
    
    async def process_ticket_with_ai(self, ticket_data: Dict) -> Dict:
        """使用AI全面处理工单"""
        ticket_id = ticket_data["id"]
        title = ticket_data["title"]
        description = ticket_data["description"]
        
        # 并行执行多个AI服务
        tasks = [
            self.classify_ticket(title, description),
            self.analyze_sentiment(f"{title} {description}"),
            self.find_similar_tickets(title, description),
            self.recommend_solutions(ticket_id, title, description, 
                                   ticket_data.get("category", ""))
        ]
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # 整合结果
        ai_analysis = {
            "ticket_id": ticket_id,
            "classification": results[0] if not isinstance(results[0], Exception) else None,
            "sentiment": results[1] if not isinstance(results[1], Exception) else None,
            "similar_tickets": results[2] if not isinstance(results[2], Exception) else None,
            "recommendations": results[3] if not isinstance(results[3], Exception) else None,
            "processing_time": time.time(),
            "confidence_score": self.calculate_overall_confidence(results)
        }
        
        # 生成综合建议
        ai_analysis["integrated_suggestions"] = await self.generate_integrated_suggestions(
            ai_analysis
        )
        
        return ai_analysis
    
    async def classify_ticket(self, title: str, description: str) -> Dict:
        """分类工单"""
        service = self.services[AIServiceType.TICKET_CLASSIFICATION]
        return await service.classify_ticket(title, description)
    
    async def analyze_sentiment(self, text: str) -> Dict:
        """分析情感"""
        service = self.services[AIServiceType.SENTIMENT_ANALYSIS]
        return await service.analyze_sentiment(text)
    
    async def find_similar_tickets(self, title: str, description: str) -> List[Dict]:
        """查找相似工单"""
        service = self.services[AIServiceType.SIMILARITY_SEARCH]
        return await service.find_similar_tickets(title, description)
    
    async def recommend_solutions(self, ticket_id: str, title: str,
                                description: str, category: str) -> Dict:
        """推荐解决方案"""
        service = self.services[AIServiceType.KNOWLEDGE_RECOMMENDATION]
        return await service.recommend_solutions(ticket_id, title, description, category)
    
    async def predict_maintenance(self, service_id: str, 
                                historical_data: List[Dict]) -> Dict:
        """预测维护"""
        service = self.services[AIServiceType.PREDICTIVE_MAINTENANCE]
        return await service.predict_failures(service_id, historical_data)
    
    def calculate_overall_confidence(self, results: List) -> float:
        """计算整体置信度"""
        confidences = []
        
        for result in results:
            if isinstance(result, dict) and "confidence" in result:
                confidences.append(result["confidence"])
        
        return sum(confidences) / len(confidences) if confidences else 0.0
    
    async def generate_integrated_suggestions(self, ai_analysis: Dict) -> List[Dict]:
        """生成综合建议"""
        suggestions = []
        
        # 基于分类结果的建议
        if ai_analysis["classification"]:
            classification = ai_analysis["classification"]
            suggestions.append({
                "type": "classification",
                "suggestion": f"建议分类为: {classification['category']}",
                "priority": classification['priority'],
                "confidence": classification['confidence']
            })
        
        # 基于情感分析的建议
        if ai_analysis["sentiment"]:
            sentiment = ai_analysis["sentiment"]
            if sentiment["sentiment"] == "negative" and sentiment["confidence"] > 0.7:
                suggestions.append({
                    "type": "priority_adjustment",
                    "suggestion": "检测到负面情绪，建议提高处理优先级",
                    "priority": "high",
                    "confidence": sentiment["confidence"]
                })
        
        # 基于相似工单的建议
        if ai_analysis["similar_tickets"]:
            similar_tickets = ai_analysis["similar_tickets"]
            if similar_tickets and similar_tickets[0]["similarity"] > 0.8:
                suggestions.append({
                    "type": "similar_resolution",
                    "suggestion": f"发现高度相似的工单，可参考解决方案",
                    "reference_ticket": similar_tickets[0]["id"],
                    "confidence": similar_tickets[0]["similarity"]
                })
        
        return suggestions
```

## 6. 监控和性能优化

### 6.1 AI 服务监控

```python
# ai_services/monitoring.py
import time
import logging
from typing import Dict, List
from dataclasses import dataclass
from prometheus_client import Counter, Histogram, Gauge

@dataclass
class AIMetrics:
    service_name: str
    request_count: int
    success_count: int
    error_count: int
    avg_response_time: float
    avg_confidence: float

class AIMonitoringService:
    def __init__(self):
        # Prometheus metrics
        self.request_counter = Counter(
            'ai_service_requests_total',
            'Total AI service requests',
            ['service', 'status']
        )
        
        self.response_time_histogram = Histogram(
            'ai_service_response_time_seconds',
            'AI service response time',
            ['service']
        )
        
        self.confidence_gauge = Gauge(
            'ai_service_confidence_score',
            'AI service confidence score',
            ['service']
        )
        
        self.error_counter = Counter(
            'ai_service_errors_total',
            'Total AI service errors',
            ['service', 'error_type']
        )
        
        self.logger = logging.getLogger(__name__)
    
    def record_request(self, service_name: str, response_time: float,
                      success: bool, confidence: float = None,
                      error_type: str = None):
        """记录请求指标"""
        # 记录请求计数
        status = 'success' if success else 'error'
        self.request_counter.labels(service=service_name, status=status).inc()
        
        # 记录响应时间
        self.response_time_histogram.labels(service=service_name).observe(response_time)
        
        # 记录置信度
        if confidence is not None:
            self.confidence_gauge.labels(service=service_name).set(confidence)
        
        # 记录错误
        if not success and error_type:
            self.error_counter.labels(service=service_name, error_type=error_type).inc()
        
        # 日志记录
        self.logger.info(
            f"AI Service: {service_name}, "
            f"Response Time: {response_time:.3f}s, "
            f"Success: {success}, "
            f"Confidence: {confidence}"
        )
    
    async def get_service_health(self) -> Dict[str, Dict]:
        """获取服务健康状态"""
        # 这里应该从实际的监控系统获取数据
        # 示例返回格式
        return {
            "ticket_classification": {
                "status": "healthy",
                "avg_response_time": 0.5,
                "success_rate": 0.95,
                "avg_confidence": 0.85
            },
            "similarity_search": {
                "status": "healthy",
                "avg_response_time": 0.3,
                "success_rate": 0.98,
                "avg_confidence": 0.78
            }
        }
```

### 6.2 缓存策略

```python
# ai_services/cache.py
import redis
import json
import hashlib
from typing import Any, Optional
from datetime import timedelta

class AICache:
    def __init__(self, redis_url: str = "redis://localhost:6379"):
        self.redis_client = redis.from_url(redis_url)
        self.default_ttl = timedelta(hours=1)
    
    def _generate_key(self, service: str, **kwargs) -> str:
        """生成缓存键"""
        key_data = f"{service}:{json.dumps(kwargs, sort_keys=True)}"
        return hashlib.md5(key_data.encode()).hexdigest()
    
    async def get(self, service: str, **kwargs) -> Optional[Any]:
        """获取缓存"""
        key = self._generate_key(service, **kwargs)
        cached_data = self.redis_client.get(key)
        
        if cached_data:
            return json.loads(cached_data)
        
        return None
    
    async def set(self, service: str, data: Any, ttl: timedelta = None, **kwargs):
        """设置缓存"""
        key = self._generate_key(service, **kwargs)
        ttl = ttl or self.default_ttl
        
        self.redis_client.setex(
            key,
            ttl,
            json.dumps(data, default=str)
        )
    
    async def invalidate(self, service: str, **kwargs):
        """清除缓存"""
        key = self._generate_key(service, **kwargs)
        self.redis_client.delete(key)
    
    async def clear_service_cache(self, service: str):
        """清除服务的所有缓存"""
        pattern = f"{service}:*"
        keys = self.redis_client.keys(pattern)
        if keys:
            self.redis_client.delete(*keys)
```

## 7. 部署和扩展

### 7.1 Docker 配置

```dockerfile
# Dockerfile.ai-services
FROM python:3.11-slim

WORKDIR /app

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# 复制依赖文件
COPY requirements.txt .

# 安装Python依赖
RUN pip install --no-cache-dir -r requirements.txt

# 复制应用代码
COPY ai_services/ ./ai_services/
COPY models/ ./models/

# 设置环境变量
ENV PYTHONPATH=/app
ENV MODEL_PATH=/app/models

# 暴露端口
EXPOSE 8000

# 启动命令
CMD ["uvicorn", "ai_services.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

### 7.2 Kubernetes 部署

```yaml
# k8s/ai-services-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-services
  labels:
    app: ai-services
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ai-services
  template:
    metadata:
      labels:
        app: ai-services
    spec:
      containers:
      - name: ai-services
        image: itsm/ai-services:latest
        ports:
        - containerPort: 8000
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379"
        - name: QDRANT_URL
          value: "http://qdrant-service:6333"
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-secrets
              key: openai-api-key
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: ai-services
spec:
  selector:
    app: ai-services
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8000
  type: ClusterIP
```

这个AI服务集成架构提供了完整的智能分析、预测维护、知识推荐等功能，通过模块化设计和微服务架构，确保了系统的可扩展性和可维护性。